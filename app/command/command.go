package command

import (
	"redis-go/app/resp"
	"strings"
)

type Command struct {
	Word string
	Args []resp.Resp
}

func NewCommand(arr []resp.Resp) *Command {
	if len(arr) > 1 {
		return nil
	}
	return &Command{
		Word: strings.ToUpper(string(arr[0].Data)),
		Args: arr[1:],
	}
}
