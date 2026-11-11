package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"redis-go/app/command"
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
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn, cache, commandDispatcher)
	}
}

func registerCommands(cd *command.CommandDispatcher) {
	cd.RegisterCommand("PING", command.Ping)
	cd.RegisterCommand("ECHO", command.Echo)
	cd.RegisterCommand("GET", command.Get)
	cd.RegisterCommand("SET", command.Set)
	cd.RegisterCommand("RPUSH", command.RPUSH)
	cd.RegisterCommand("LRANGE", command.LRANGE)
}

func handleConnection(conn net.Conn, cache *storage.Cache, dispatcher *command.CommandDispatcher) {
	defer conn.Close()
	parser := parser.NewParser(conn)
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
		conn.Write(dispatcher.DispatchCommand(*c, cache))
	}
}
