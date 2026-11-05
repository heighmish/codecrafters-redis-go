package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type Type rune

const (
	Array   = '*'
	Bulk    = '$'
	Error   = '-'
	Integer = ':'
	String  = '+'
)

type Resp struct {
	Type Type
	Data []byte
}

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn)
	}
}

func getType(s string) Type {
	switch s[0] {
	case Bulk:
		return Bulk
	case Integer:
		return Integer
	case String:
		return String
	case Array:
		return Array
	case Error:
		return Error
	default:
		fmt.Println("Unknown type: ", s[0])
		panic("Unknown type")
	}
}

func parseInteger(s string) Resp {
	return Resp{
		Type: Integer,
		Data: []byte(s[1:]),
	}
}

func parseBulkString(_ string, val string) Resp {
	return Resp{
		Type: Bulk,
		Data: []byte(val),
	}
}

func (r *Resp) ToBulkString() string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(r.Data), r.Data)
}

func handleConnection(conn net.Conn) {
	buf := make([]byte, 512)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading from connection: ", err.Error())
			conn.Close()
			return
		}
		/* *2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n
		 * = [*2, $4, ECHO, $3, hey]
		 */
		commands := strings.Split(string(buf), "\r\n")
		arr := make([]Resp, 0)
		for i := 0; i < len(commands); {
			respType := getType(commands[i])
			var resp Resp
			switch respType {
			case Integer:
				resp = parseInteger(commands[i])
				i += 1
			case String:
				resp = parseBulkString(commands[i], commands[i+1])
				i += 2
			default:
				i += 1
			}
			arr = append(arr, resp)
		}

		command := arr[0]
		switch strings.ToLower(string(command.Data)) {
		case "ping":
			conn.Write([]byte("+PONG\r\n"))
		case "echo":
			echo := make([]string, len(arr[1:]))
			for i, resp := range arr[1:] {
				echo[i] = resp.ToBulkString()
			}
			conn.Write(strings.Join(echo, "\r\n"))
		default:
			fmt.Println("Unknown command: ", command)
			continue
		}

	}
}
