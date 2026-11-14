package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"redis-go/app/command"
	connContext "redis-go/app/context"
	"redis-go/app/parser"
	"redis-go/app/storage"
)

func main() {
	fmt.Println("Logs from your program will appear here!")
	commandDispatcher := command.NewCommandDispatcher()
	registerCommands(commandDispatcher)

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()
	cache := storage.NewCache()
	id := 0
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn, cache, commandDispatcher, id)
		id += 1
	}
}

func registerCommands(cd *command.CommandDispatcher) {
	cd.RegisterCommand("PING", command.Ping)
	cd.RegisterCommand("ECHO", command.Echo)
	cd.RegisterCommand("GET", command.Get)
	cd.RegisterCommand("SET", command.Set)
	cd.RegisterCommand("RPUSH", command.RPUSH)
	cd.RegisterCommand("LRANGE", command.LRANGE)
	cd.RegisterCommand("LPUSH", command.LPUSH)
	cd.RegisterCommand("LLEN", command.LLEN)
	cd.RegisterCommand("LPOP", command.LPOP)
	cd.RegisterCommand("BLPOP", command.BLPOP)
	cd.RegisterCommand("TYPE", command.TYPE)
	cd.RegisterCommand("XADD", command.XADD)
}

func handleConnection(conn net.Conn, cache *storage.Cache, dispatcher *command.CommandDispatcher, id int) {
	defer conn.Close()
	parser := parser.NewParser(conn)
	context := connContext.NewContext(id)
	for {
		fmt.Println("Waiting for next command...")
		value, err := parser.ReadResp()
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed...")
				return
			}

			fmt.Printf("Error reading RESP: %v", err)
			return
		}

		fmt.Println("Successfully received next command!")
		c := command.NewCommand(value.Elements)
		conn.Write(dispatcher.DispatchCommand(*c, cache, context))
	}
}
