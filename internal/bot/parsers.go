package bot

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type setRoomCommand struct {
	zoneName    string
	mode        string
	temperature float64
	duration    time.Duration
}

func parseSetRoom(args ...string) (setRoomCommand, error) {
	if len(args) < 2 {
		return setRoomCommand{}, fmt.Errorf("missing parameters\nUsage: set room <room> [auto|<temperature> [<duration>]")
	}

	cmd := setRoomCommand{
		zoneName: args[0],
		mode:     args[1],
	}

	if cmd.mode == "auto" {
		return cmd, nil
	}

	var err error
	if cmd.temperature, err = strconv.ParseFloat(args[1], 32); err != nil {
		return setRoomCommand{}, fmt.Errorf("invalid target temperature: %q", args[1])
	}

	if len(args) == 2 {
		return cmd, nil
	}

	if cmd.duration, err = time.ParseDuration(args[2]); err != nil {
		return setRoomCommand{}, fmt.Errorf("invalid duration: %q", args[2])
	}

	return cmd, nil

}

type setHomeCommand struct{}

func parseSetHome(args ...string) (setHomeCommand, error) {
	panic("TODO")
}

func tokenizeText(input string) []string {
	cleanInput := input
	for _, quote := range []string{"“", "”", "'"} {
		cleanInput = strings.ReplaceAll(cleanInput, quote, "\"")
	}
	r := regexp.MustCompile(`[^\s"]+|"([^"]*)"`)
	output := r.FindAllString(cleanInput, -1)

	for index, word := range output {
		output[index] = strings.Trim(word, "\"")
	}
	return output
}
