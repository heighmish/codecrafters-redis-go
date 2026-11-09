package command

import (
	"fmt"
	"redis-go/app/resp"
	"strings"
)

type Command struct {
	Word string
	Args []resp.Resp
}

func NewCommand(arr []resp.Resp) *Command {
	if len(arr) < 1 {
		return &Command{
			Word: "ERR",
			Args: make([]resp.Resp, 0),
		}
	}
	return &Command{
		Word: strings.ToUpper(string(arr[0].Data)),
		Args: arr[1:],
	}
}

func ExecuteCommand(c *Command) []byte {
	fmt.Println("Processing command: ", c.Word)
	switch c.Word {
	case "PING":
		return ([]byte("+PONG\r\n"))
	case "ECHO":
		if len(c.Args) < 1 {
			return []byte("-ERR Usage: echo <arg>\r\n")
		}

		return ([]byte(c.Args[0].ToBulkString()))
	default:
		fmt.Println("Unknown command: ", c.Word)
		return ([]byte("-ERR unknown command\r\n"))
	}
}
