package tadobot

import (
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

// <@U01N8HG45JQ> set "living room” auto
func parseText(input string) (output []string) {
	cleanInput := strings.Replace(input, "“", "\"", -1)
	cleanInput = strings.Replace(cleanInput, "”", "\"", -1)
	r := regexp.MustCompile(`[^\s"]+|"([^"]*)"`)
	output = r.FindAllString(cleanInput, -1)

	log.WithField("parsed", output).Debug("parsed slack input")
	for index, word := range output {
		output[index] = strings.Trim(word, "\"")
	}
	return
}

func (bot *TadoBot) parseCommand(input string) (command string, args []string) {
	words := parseText(input)
	if len(words) > 1 && words[0] == "<@"+bot.userID+">" {
		command = words[1]
		args = words[2:]
	}

	return
}
