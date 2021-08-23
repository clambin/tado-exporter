package slackbot

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseText(t *testing.T) {
	var output []string

	output = parseText("Hello world")
	if assert.Len(t, output, 2) {
		assert.Equal(t, "Hello", output[0])
		assert.Equal(t, "world", output[1])
	}

	output = parseText("He said \"Hello world\"")
	if assert.Len(t, output, 3) {
		assert.Equal(t, "He", output[0])
		assert.Equal(t, "said", output[1])
		assert.Equal(t, "Hello world", output[2])
	}

	output = parseText("")
	assert.Len(t, output, 0)

	output = parseText("\"Hello world\"")
	if assert.Len(t, output, 1) {
		assert.Equal(t, "Hello world", output[0])
	}

	output = parseText("\"\"")
	if assert.Len(t, output, 1) {
		assert.Equal(t, "", output[0])
	}

	output = parseText("\"")
	assert.Len(t, output, 0)
}

func TestParseCommand(t *testing.T) {
	var (
		command string
		args    []string
	)

	bot := Agent{
		userID: "123",
	}

	command, args = bot.parseCommand("")
	assert.Equal(t, "", command)
	assert.Len(t, args, 0)

	command, args = bot.parseCommand("hello world")
	assert.Equal(t, "", command)
	assert.Len(t, args, 0)

	command, args = bot.parseCommand("<@123> version")
	assert.Equal(t, "version", command)
	assert.Len(t, args, 0)

	command, args = bot.parseCommand("<@123> set room 18.0")
	assert.Equal(t, "set", command)
	if assert.Len(t, args, 2) {
		assert.Equal(t, "room", args[0])
		assert.Equal(t, "18.0", args[1])
	}

	command, args = bot.parseCommand("<@123> set \"the lobby\" 18.0")
	assert.Equal(t, "set", command)
	if assert.Len(t, args, 2) {
		assert.Equal(t, "the lobby", args[0])
		assert.Equal(t, "18.0", args[1])
	}

	command, args = bot.parseCommand("<@123> set “the lobby“ 18.0")
	assert.Equal(t, "set", command)
	if assert.Len(t, args, 2) {
		assert.Equal(t, "the lobby", args[0])
		assert.Equal(t, "18.0", args[1])
	}

}
