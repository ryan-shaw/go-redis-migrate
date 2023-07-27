package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"sync/atomic"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

var counters = make(map[string]*int64)
var writeCommands = make(map[string]struct{})

// Check if the passed command is write command
func isWriteCommand(cmd string) bool {
	_, ok := writeCommands[strings.ToLower(cmd)]
	return ok
}

// parseCommand takes a command string as input and returns a slice of strings,
// where each string in the slice represents a part of the command.
//
// The function splits the incoming command by the double quotes character (`"`),
// then trims each part of unnecessary leading and trailing whitespace.
// It omits any parts that are empty or contain only a single space.
func parseCommand(cmd string) []string {
	parts := strings.Split(cmd, "\"")
	var parsedParts []string
	for _, part := range parts {
		part = strings.Trim(part, " ")
		if part != "" && part != " " {
			parsedParts = append(parsedParts, part)
		}
	}
	return parsedParts
}

const numWorkers = 50

// worker is a goroutine function designed to process Redis commands.
// It continuously receives command strings from the provided channel (ch),
// parses them into an array of strings, and checks if the parsed command is a write command.
// If the command is a write command, it converts the command arguments to an interface slice and
// passes them to the Redis client (dstRdb) to perform the operation.
//
// The function also updates a global counter for the executed command.
// This is done in a thread-safe manner using atomic operations.
func worker(ch <-chan string, dstRdb *redis.Client) {
	for cmd := range ch {
		parsedCommand := parseCommand(cmd)
		if len(parsedCommand) > 1 && isWriteCommand(parsedCommand[1]) {
			interfaceSlice := make([]interface{}, len(parsedCommand[1:]))
			for i, v := range parsedCommand[1:] {
				interfaceSlice[i] = v
			}
			dstRdb.Do(context.Background(), interfaceSlice...)

			commandName := parsedCommand[1]
			counter, ok := counters[commandName]
			if !ok {
				var newCounter int64
				counter = &newCounter
				counters[commandName] = counter
			}
			atomic.AddInt64(counter, 1)
		}
	}
}

func getWriteCommands(host string) {
	rdb := redis.NewClient(&redis.Options{
		Addr: host,
		DB:   0,
	})

	ctx := context.Background()
	cmd := rdb.Do(ctx, "COMMAND")
	cmds, err := cmd.Result()
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	var writeCmds []string

	for _, cmdInfo := range cmds.([]interface{}) {
		cmdDetail := cmdInfo.([]interface{})
		cmdName := cmdDetail[0].(string)
		flags := cmdDetail[2].([]interface{})

		for _, flag := range flags {
			if flag.(string) == "write" {
				writeCmds = append(writeCmds, cmdName)
			}
		}
	}

	log.Info("Write commands: ", strings.Join(writeCmds, ", "))
	for _, cmd := range writeCmds {
		writeCommands[cmd] = struct{}{}
	}
}

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	var sourceHost string
	flag.StringVar(&sourceHost, "sourceHost", "localhost:6379", "The host of the source Redis data")

	var targetHost string
	flag.StringVar(&targetHost, "targetHost", "localhost:6380", "The host of the target Redis data")

	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")

	flag.Parse()
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// Set up a new Redis client for destination instance
	dstRdb := redis.NewClient(&redis.Options{
		Addr:     targetHost,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	conn, err := net.Dial("tcp", sourceHost)
	defer conn.Close()

	buf := bufio.NewReader(conn)
	getWriteCommands(sourceHost)

	log.Println(sourceHost, targetHost)

	if err != nil {
		log.Fatal(err)
	}

	// Create a channel to distribute commands
	ch := make(chan string)

	// Start multiple worker goroutines
	for i := 0; i < numWorkers; i++ {
		go worker(ch, dstRdb)
	}

	// Stats log goroutine
	go func() {
		var overall int64
		for {
			var total int64
			var fields = log.Fields{}
			for commandName, counter := range counters {
				count := atomic.SwapInt64(counter, 0)
				fields[commandName] = count
				total += count
			}
			overall += total
			fields["overall"] = overall
			log.WithFields(fields).Infof("Processed %d total commands in the last second", total)

			time.Sleep(1 * time.Second)
		}
	}()

	// Wait to target host to become master
	var writeable = false
	go func() {
		for {
			// This will hammer the target with info requests every ms to get quickest update possible
			// but will stop as soon as "master" is seen
			infoCmd := dstRdb.Info(context.Background(), "replication")
			info, err := infoCmd.Result()

			if err != nil {
				log.Fatal(err)
			}
			lines := strings.Split(info, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "role:") {
					role := strings.TrimSpace(line[5:])
					if role == "master" {
						writeable = true
					}
				}
			}
			if writeable {
				log.Warn("Target is master - starting writes")
				break
			}
			time.Sleep(1)
		}
	}()

	_, err = conn.Write([]byte("MONITOR\r\n"))
	if err != nil {
		log.Fatal(err)
	}
	for {
		str, err := buf.ReadString('\n')
		if !writeable {
			continue
		}
		if len(str) > 0 {
			if str == "+OK\r\n" {
				log.Debug("OK")
			}
			if err != nil {
				log.Fatal(err)
			}
			str = strings.TrimSuffix(str, "\r\n")
			ch <- str
		}
	}
}
