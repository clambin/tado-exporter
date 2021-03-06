package commands

type RequestChannel chan Command
type ResponseChannel chan []string

type Command struct {
	Command  int
	Response ResponseChannel
}

const Report = 1
