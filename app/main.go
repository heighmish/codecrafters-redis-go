package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
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
	T      Type
	Data   []byte
	Length int
}

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

func (r *Resp) ToBulkString() string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(r.Data), r.Data)
}

type Parser struct {
	reader *bufio.Reader
}

func NewParser(io io.Reader) Parser {
	return Parser{
		reader: bufio.NewReader(io),
	}
}

func (parser *Parser) ReadLine() (string, error) {
	line, err := parser.reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read: ", err)
		return "", err
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

func (parser *Parser) ReadInt() (int, error) {
	line, err := parser.ReadLine()
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(line)
	if err != nil {
		fmt.Println("[ReadInt] Error parsing: ", line)
		return 0, err
	}
	return value, nil
}

func (parser *Parser) ReadResp() (Resp, error) {
	t, err := parser.reader.ReadByte()
	if err != nil {
		fmt.Println("Error parsing type: ", err)
		return Resp{}, err
	}
	//fmt.Printf("Byte: %q | Hex: 0x%02X\n", t, t)
	switch t {
	case Integer:
		fmt.Println("Integer: ", string(t))
		line, err := parser.ReadLine()
		if err != nil {
			return Resp{}, err
		}
		return Resp{T: Integer, Data: []byte(line)}, nil
	case Bulk:
		fmt.Println("BulkString: ", string(t))
		strLength, err := parser.ReadInt()
		if err != nil {
			return Resp{}, err
		}

		bulkString := make([]byte, strLength)
		_, err = io.ReadFull(parser.reader, bulkString)
		if err != nil {
			return Resp{}, err
		}
		// Consume trailing \r\n
		parser.ReadLine()

		return Resp{T: Bulk, Data: []byte(bulkString)}, nil
	case Array:
		fmt.Println("Array: ", string(t))
		length, err := parser.ReadInt()
		if err != nil {
			panic("Array length is fake")
		}
		return Resp{T: Array, Length: length}, nil
	case Error:
		parser.ReadLine()
		return Resp{T: Error}, nil
	case String:
		fmt.Println("Simple string: ", string(t))
		line, err := parser.ReadLine()
		if err != nil {
			return Resp{}, err
		}
		return Resp{T: String, Data: []byte(line)}, nil
	default:
		fmt.Println("Unknown Type: ", string(t))
		return Resp{}, err
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	parser := NewParser(conn)
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
		if value.T == Array && value.Length > 0 {
			commandArray := make([]Resp, value.Length)
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
