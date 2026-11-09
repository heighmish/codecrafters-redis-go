package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"redis-go/app/parser"
	"redis-go/app/resp"
	"strings"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
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
		/* *2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n
		 * = [*2, $4, ECHO, $3, hey]
		 */
		fmt.Printf("Received resp: Type: %s, val: %s, length: %d\r\n", string(value.T), string(value.Data), value.Length)
		if value.T == resp.Array && value.Length > 0 {
			commandArray := make([]resp.Resp, value.Length)
			for i := range value.Length {
				value, err := parser.ReadResp()
				if err != nil {
					fmt.Printf("Error reading RESP: %v", err)
					return
				}
				commandArray[i] = value
			}
			command := strings.ToUpper(string(commandArray[0].Data))
			fmt.Println("Processing command: ", command)
			switch command {
			case "PING":
				conn.Write([]byte("+PONG\r\n"))
			case "ECHO":
				conn.Write([]byte(commandArray[1].ToBulkString()))
			default:
				fmt.Println("Unknown command: ", command)
				conn.Write([]byte("-ERR unknown command\r\n"))
			}
		}
	}
}
