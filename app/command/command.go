package command

import (
	"fmt"
	"redis-go/app/resp"
	"redis-go/app/storage"
	"strconv"
	"strings"
	"time"
)

type CommandDispatcher struct {
	commands map[string]func([]*resp.Resp, *storage.Cache) []byte
}

func NewCommandDispatcher() *CommandDispatcher {
	return &CommandDispatcher{
		commands: make(map[string]func([]*resp.Resp, *storage.Cache) []byte),
	}
}

func (cd *CommandDispatcher) RegisterCommand(command string, f func([]*resp.Resp, *storage.Cache) []byte) {
	cd.commands[command] = f
}

func (cd *CommandDispatcher) DispatchCommand(command Command, cache *storage.Cache) []byte {
	f, ok := cd.commands[command.Name]
	if !ok {
		fmt.Println("Unknown command: ", command)
		return ([]byte("-ERR unknown command\r\n"))
	}

	return f(command.Args, cache)
}

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

func Ping(args []*resp.Resp, cache *storage.Cache) []byte {
	return []byte(resp.ToSimpleString("PONG"))
}

func Echo(args []*resp.Resp, cache *storage.Cache) []byte {
	if len(args) < 1 {
		return []byte("-ERR Usage: ECHO <arg>\r\n")
	}

	return ([]byte(args[0].ToBulkString()))
}

func Set(args []*resp.Resp, cache *storage.Cache) []byte {
	if len(args) < 2 {
		return []byte("-ERR Usage: SET <key> <value>\r\n")
	}

	key := args[0].Data

	if len(args) == 4 {
		opt := args[2]
		if strings.ToUpper(opt.Data) == "PX" {
			expiry, err := strconv.Atoi(args[3].Data)
			if err != nil {
				return []byte("-ERR Usage: SET <key> <value> <px> <expiryInMilliseconds>\r\n")
			}
			expiryTime := time.Now().UTC().Add(time.Duration(expiry) * time.Millisecond)
			cache.Container.Add(key, storage.NewValueWithExpiry(*args[1], &expiryTime))
			fmt.Printf("Adding key %v with expiry time %v\r\n", key, expiryTime)
			return []byte(resp.ToSimpleString("OK"))
		}
	}
	cache.Container.Add(key, storage.NewValue(*args[1]))

	return []byte(resp.ToSimpleString("OK"))

}

func Get(args []*resp.Resp, cache *storage.Cache) []byte {
	if len(args) < 1 {
		return []byte(resp.ToError("Usage: GET <key>"))
	}
	key := args[0].Data

	val, ok := cache.Container.Get(key)
	if !ok {
		fmt.Printf("Key %v not found\r\n", key)
		return []byte(resp.ToBulkString(""))
	}

	return []byte(resp.ToBulkString(val.Data))
}

func RPUSH(args []*resp.Resp, cache *storage.Cache) []byte {
	if len(args) < 2 {
		return []byte(resp.ToError("Usage: RPUSH <list_key> <value_1> <value_...> <value_n>"))
	}

	key := args[0].Data
	length := cache.Lists.Push(key, args[1:])
	return []byte(resp.ToRespInt(length))
}

func LPUSH(args []*resp.Resp, cache *storage.Cache) []byte {
	if len(args) < 2 {
		return []byte(resp.ToError("Usage: LPUSH <list_key> <value_1> <value_...> <value_n>"))
	}

	key := args[0].Data
	length := cache.Lists.Prepend(key, args[1:])
	return []byte(resp.ToRespInt(length))
}

func normaliseIndex(val, len int) int {
	if val < 0 {
		if val*-1 > len {
			return 0
		}
		return len + val
	} else {
		if val > len {
			return len
		}
		return val
	}
}

func LRANGE(args []*resp.Resp, cache *storage.Cache) []byte {
	if len(args) < 3 {
		return []byte(resp.ToError("Usage: LRANGE <list_key> <start_index> <stop_index>"))
	}
	startIdx, err := strconv.Atoi(args[1].Data)
	if err != nil {
		return []byte(resp.ToError("Usage: LRANGE <list_key> <start_index> <stop_index>"))
	}

	stopIdx, err := strconv.Atoi(args[2].Data)
	if err != nil {
		return []byte(resp.ToError("Usage: LRANGE <list_key> <start_index> <stop_index>"))
	}

	key := args[0].Data
	arr, ok := cache.Lists.GetList(key)
	if !ok {
		fmt.Printf("Array with key %v not found\r\n", key)
	}

	startIdx = normaliseIndex(startIdx, len(arr))
	stopIdx = normaliseIndex(stopIdx, len(arr))

	if startIdx >= len(arr) || startIdx > stopIdx {
		return []byte("*0\r\n")
	}

	fmt.Printf("Array Length: %v, startIdx: %v, stopIdx: %v\r\n", len(arr), startIdx, stopIdx)
	return []byte(resp.ToRespArray(arr[startIdx:min(stopIdx+1, len(arr))]))

}
