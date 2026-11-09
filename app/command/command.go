package command

import (
	"fmt"
	"redis-go/app/resp"
	"strings"
)

type Command struct {
	Name string
	Args []*resp.Resp
}

func NewCommand(arr []*resp.Resp) *Command {
	if len(arr) < 1 {
		return &Command{
			Name: "ERR",
			Args: make([]*resp.Resp, 0),
		}
	}
	return &Command{
		Name: strings.ToUpper(string(arr[0].Data)),
		Args: arr[1:],
	}
}

func ExecuteCommand(c *Command) []byte {
	fmt.Println("Processing command: ", c.Name)
	switch c.Name {
	case "PING":
		return ([]byte("+PONG\r\n"))
	case "ECHO":
		if len(c.Args) < 1 {
			return []byte("-ERR Usage: echo <arg>\r\n")
		}

		return ([]byte(c.Args[0].ToBulkString()))
	default:
		fmt.Println("Unknown command: ", c.Name)
		return ([]byte("-ERR unknown command\r\n"))
	}
}
