package tadobot

import "regexp"

func parseText(input string) (output []string) {
	r := regexp.MustCompile(`[^\s"]+|"([^"]*)"`)
	output = r.FindAllString(input, -1)
	for index, word := range output {
		length := len(word)
		if word[0] == '"' && word[length-1] == '"' {
			output[index] = word[1 : length-1]
		}
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
