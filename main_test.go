package main

import (
	"testing"
)

func TestParseCommand(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"set \"key\" \"value\"", []string{"set", "key", "value"}},
		{"get \"key\"", []string{"get", "key"}},
		{"  mget   \"key1\"  \"key2\"  \"key3\"  ", []string{"mget", "key1", "key2", "key3"}},
		{"hmset \"user:123\" \"name\" \"John\" \"age\" 30", []string{"hmset", "user:123", "name", "John", "age", "30"}},
		{"hgetall \"user:123\"", []string{"hgetall", "user:123"}},
		{"", []string{}}, // Test with empty input string
		{"\"commandWithNoSpaces\"", []string{"commandWithNoSpaces"}},    // Test with command containing no spaces
		{"get \"key with spaces\"", []string{"get", "key with spaces"}}, // Test with command containing spaces in the argument
	}

	for _, tc := range testCases {
		result := parseCommand(tc.input)

		if len(result) != len(tc.expected) {
			t.Errorf("Input: %q, Expected: %v, Got: %v", tc.input, tc.expected, result)
			continue
		}

		for i := 0; i < len(result); i++ {
			if result[i] != tc.expected[i] {
				t.Errorf("Input: %q, Expected: %v, Got: %v", tc.input, tc.expected, result)
				break
			}
		}
	}
}

func TestIsWriteCommand(t *testing.T) {
	writeCommands = map[string]struct{}{
		"set":    struct{}{},
		"hmset":  struct{}{},
		"hset":   struct{}{},
		"lpush":  struct{}{},
		"zadd":   struct{}{},
		"sadd":   struct{}{},
		"append": struct{}{},
	}

	testCases := []struct {
		command  string
		expected bool
	}{
		{"SET", true},      // Should be case-insensitive
		{"set", true},      // Should be case-insensitive
		{"HSET", true},     // Should be case-insensitive
		{"Get", false},     // Non-write command
		{"LPUSH", true},    // Should be case-insensitive
		{"ZADD", true},     // Should be case-insensitive
		{"DEL", false},     // Non-write command
		{"APPEND", true},   // Should be case-insensitive
		{"hgetall", false}, // Non-write command
	}

	for _, tc := range testCases {
		result := isWriteCommand(tc.command)

		if result != tc.expected {
			t.Errorf("Command: %s, Expected: %v, Got: %v", tc.command, tc.expected, result)
		}
	}
}
