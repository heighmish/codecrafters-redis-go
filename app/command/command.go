package command

import (
	"fmt"
	"redis-go/app/resp"
	"redis-go/app/storage"
	"strconv"
	"strings"
	"time"
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

func ExecuteCommand(c *Command, container *storage.Container, lists *storage.ListMap) []byte {
	fmt.Println("Processing command: ", c.Name)
	switch c.Name {
	case "PING":
		return []byte(resp.ToSimpleString("PONG"))
	case "ECHO":
		if len(c.Args) < 1 {
			return []byte("-ERR Usage: ECHO <arg>\r\n")
		}

		return ([]byte(c.Args[0].ToBulkString()))
	case "SET":
		if len(c.Args) < 2 {
			return []byte("-ERR Usage: SET <key> <value>\r\n")
		}

		key := c.Args[0].Data

		if len(c.Args) == 4 {
			opt := c.Args[2]
			if strings.ToUpper(opt.Data) == "PX" {
				expiry, err := strconv.Atoi(c.Args[3].Data)
				if err != nil {
					return []byte("-ERR Usage: SET <key> <value> <px> <expiryInMilliseconds>\r\n")
				}
				expiryTime := time.Now().UTC().Add(time.Duration(expiry) * time.Millisecond)
				container.Add(key, storage.NewValueWithExpiry(*c.Args[1], &expiryTime))
				fmt.Printf("Adding key %v with expiry time %v\r\n", key, expiryTime)
				return []byte(resp.ToSimpleString("OK"))
			}
		}
		container.Add(key, storage.NewValue(*c.Args[1]))

		return []byte(resp.ToSimpleString("OK"))
	case "GET":
		if len(c.Args) < 1 {
			return []byte(resp.ToError("Usage: GET <key>"))
		}
		key := c.Args[0].Data

		val, ok := container.Get(key)
		if !ok {
			fmt.Printf("Key %v not found\r\n", key)
			return []byte(resp.ToBulkString(""))
		}

		return []byte(resp.ToBulkString(val.Data))
	case "RPUSH":
		if len(c.Args) < 2 {
			return []byte(resp.ToError("Usage: RPUSH <list_key> <value_1> <value_...> <value_n>"))
		}

		key := c.Args[0].Data
		length := lists.Push(key, c.Args[1:])
		return []byte(resp.ToRespInt(length))
	case "LRANGE":
		if len(c.Args) < 3 {
			return []byte(resp.ToError("Usage: LRANGE <list_key> <start_index> <stop_index>"))
		}
		startIdx, err := strconv.Atoi(c.Args[1].Data)
		if err != nil {
			return []byte(resp.ToError("Usage: LRANGE <list_key> <start_index> <stop_index>"))
		}

		stopIdx, err := strconv.Atoi(c.Args[2].Data)
		if err != nil {
			return []byte(resp.ToError("Usage: LRANGE <list_key> <start_index> <stop_index>"))
		}

		key := c.Args[0].Data
		arr, ok := lists.GetList(key)
		if !ok {
			fmt.Printf("Array with key %v not found\r\n", key)
		}
		if startIdx >= len(arr) || startIdx > stopIdx {
			return []byte("*0\r\n")
		}

		return []byte(resp.ToRespArray(arr[startIdx:min(stopIdx+1, len(arr))]))

	default:
		fmt.Println("Unknown command: ", c.Name)
		return ([]byte("-ERR unknown command\r\n"))
	}
}
