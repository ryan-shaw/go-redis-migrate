# 🚀 Redis command replicator 🚀

This program is a Redis command processor that reads commands from a source Redis instance, processes them, and then writes them to a target Redis instance. It's a multi-threaded application that uses a worker-pool pattern for processing commands concurrently. This program also keeps track of the number of executed commands for performance monitoring.

## 📝 Features 📝
- Reads commands from a source Redis instance
- Waits for target to becomem writeable (role:master)
- Processes commands and writes them to a target Redis instance
- Multi-threaded operation using a worker-pool pattern for processing commands concurrently
- Performance monitoring with a command counter

## 🛠️ How to use 🛠️
You can customize the source and target Redis hosts via command-line flags. There is also a debug mode flag to increase logging verbosity.

Example usage:

```sh
redis-migrate -sourceHost "localhost:6379" -targetHost "localhost:6380" -debug
```

## 📈 Performance Monitoring 📈
The program keeps track of the number of executed commands in a thread-safe manner. It uses atomic operations to update a global counter. The counters are logged every second to monitor the system's performance.

## 🚨 Warning 🚨

As specified in the Redis docs running `MONITOR` can have serious impact on performance, up to 50% performance hit depending on the workload. The [benchmark](https://redis.io/commands/monitor/) exmaple is quite extreme as that is maxing the capabilities of Redis and hopefully your normal workload is not near 100% resource usage. I would seriously recommend ensuring their is enough resource capacity on your source Redis instance before running this program.

It would be better to have this as a proxy to then proxy to both Redis instance but I want to have this without any modification to the stack.
