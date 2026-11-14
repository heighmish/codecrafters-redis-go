package command

import (
	"fmt"
	connContext "redis-go/app/context"
	"redis-go/app/resp"
	"redis-go/app/storage"
	"strconv"
	"strings"
	"time"
)

type DispatchFunction func([]resp.Resp, *storage.Cache, *connContext.Context) []byte

type CommandDispatcher struct {
	commands map[string]DispatchFunction
}

func NewCommandDispatcher() *CommandDispatcher {
	return &CommandDispatcher{
		commands: make(map[string]DispatchFunction),
	}
}

func (cd *CommandDispatcher) RegisterCommand(command string, f DispatchFunction) {
	cd.commands[command] = f
}

func (cd *CommandDispatcher) DispatchCommand(command Command, cache *storage.Cache, ctx *connContext.Context) []byte {
	f, ok := cd.commands[command.Name]
	if !ok {
		fmt.Println("Unknown command: ", command)
		return ([]byte("-ERR unknown command\r\n"))
	}

	return f(command.Args, cache, ctx)
}

type Command struct {
	Name string
	Args []resp.Resp
}

func NewCommand(arr []resp.Resp) *Command {
	if len(arr) < 1 {
		return &Command{
			Name: "ERR",
		}
	}
	return &Command{
		Name: strings.ToUpper(arr[0].Data),
		Args: arr[1:],
	}
}

func Ping(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
	return []byte(resp.ToSimpleString("PONG"))
}

func Echo(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
	if len(args) < 1 {
		return []byte("-ERR Usage: ECHO <arg>\r\n")
	}

	return ([]byte(args[0].ToBulkString()))
}

func Set(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
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
			cache.Container.Add(key, storage.NewValueWithExpiry(args[1], &expiryTime))
			fmt.Printf("Adding key %v with expiry time %v\r\n", key, expiryTime)
			return []byte(resp.ToSimpleString("OK"))
		}
	}
	cache.Container.Add(key, storage.NewValue(args[1]))

	return []byte(resp.ToSimpleString("OK"))

}

func Get(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
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

func RPUSH(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
	if len(args) < 2 {
		return []byte(resp.ToError("Usage: RPUSH <list_key> <value_1> <value_...> <value_n>"))
	}

	key := args[0].Data
	length := cache.Lists.Push(key, args[1:])
	return []byte(resp.ToRespInt(length))
}

func LPUSH(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
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

func LRANGE(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
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

func LLEN(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
	if len(args) != 1 {
		return []byte(resp.ToError("Usage: LLEN <list_key>"))
	}

	key := args[0].Data
	return []byte(resp.ToRespInt(cache.Lists.Len(key)))
}

func LPOP(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
	if len(args) > 2 {
		return []byte(resp.ToError("Usage: LPOP <list_key>"))
	}

	key := args[0].Data
	if len(args) == 2 {
		removalCount, err := strconv.Atoi(args[1].Data)
		if err != nil || removalCount < 0 {
			return []byte(resp.ToError("Usage: LPOP <list_key> <num_to_pop>"))
		}
		slice, ok := cache.Lists.PopElements(key, removalCount)

		if !ok {
			return []byte(resp.ToBulkString(""))
		}

		return []byte(resp.ToRespArray(slice))
	}

	slice, ok := cache.Lists.PopElements(key, 1)
	if !ok {
		return []byte(resp.ToBulkString(""))
	}

	return []byte(slice[0].String())
}

func BLPOP(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
	// Todo!
	if len(args) > 2 {
		return []byte(resp.ToError("Usage: BLPOP <list_key> <timeout>"))
	}

	timeout := 0.0
	if len(args) == 2 {
		timeoutParsed, err := strconv.ParseFloat(args[1].Data, 64)
		if err == nil {
			timeout = timeoutParsed
		}
	}

	key := args[0].Data
	val, ok := cache.Lists.Blpop(key, timeout, ctx.Id)
	if !ok {
		return []byte("*-1\r\n")
	}

	arr := make([]resp.Resp, 2)
	arr[0] = args[0]
	arr[1] = val

	return []byte(resp.ToRespArray(arr))
}

func TYPE(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
	if len(args) != 1 {
		return []byte(resp.ToError("Usage: TYPE <key>"))
	}
	key := args[0].Data
	_, ok := cache.Container.Get(key)
	if ok {
		return []byte(resp.ToSimpleString("string"))
	}

	_, ok = cache.Streams.GetStream(key)
	if ok {
		return []byte(resp.ToSimpleString("stream"))
	}

	return []byte(resp.ToSimpleString("none"))
}

func XADD(args []resp.Resp, cache *storage.Cache, ctx *connContext.Context) []byte {
	if len(args) < 4 {
		// Todo! generate ids if missing this will break
		return []byte(resp.ToError("Usage: XADD <stream_key> <entry_id> <key_1> <value_1>"))
	}

	key := args[0].Data
	id, err := cache.Streams.AddEntry(key, args[1].Data, args[2:])
	if err != nil {
		fmt.Println(err)
		return []byte(resp.ToError(err.Error()))
	}

	return []byte(resp.ToBulkString(id))
}
