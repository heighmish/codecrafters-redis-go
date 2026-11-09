package parser

import (
	"bufio"
	"fmt"
	"io"
	"redis-go/app/resp"
	"strconv"
	"strings"
)

type Parser struct {
	reader *bufio.Reader
}

func NewParser(io io.Reader) *Parser {
	return &Parser{
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

func (parser *Parser) ReadResp() (resp.Resp, error) {
	t, err := parser.reader.ReadByte()
	if err != nil {
		fmt.Println("Error parsing type: ", err)
		return resp.Resp{}, err
	}
	//fmt.Printf("Byte: %q | Hex: 0x%02X\n", t, t)
	switch t {
	case resp.Integer:
		return parser.parseInt()
	case resp.Bulk:
		return parser.parseBulkString()
	case resp.Array:
		return parser.parseArray()
	case resp.Error:
		return parser.parseError()
	case resp.String:
		return parser.parseString()
	default:
		fmt.Println("Unknown Type: ", string(t))
		return resp.Resp{}, err
	}
}

func (p *Parser) parseInt() (resp.Resp, error) {
	line, err := p.ReadLine()
	if err != nil {
		return resp.Resp{}, err
	}
	value, err := strconv.Atoi(line)
	if err != nil {
		fmt.Println("[ParseInt] Error parsing: ", line)
		return resp.Resp{}, err
	}
	return resp.Resp{T: resp.Integer, Integer: value}, nil

}

func (p *Parser) parseBulkString() (resp.Resp, error) {
	val, err := p.parseInt()
	if err != nil {
		return resp.Resp{}, err
	}

	bulkString := make([]byte, val.Integer)
	_, err = io.ReadFull(p.reader, bulkString)
	if err != nil {
		return resp.Resp{}, err
	}
	// Consume trailing \r\n
	p.ReadLine()

	return resp.Resp{T: resp.Bulk, Data: string(bulkString)}, nil
}

func (p *Parser) parseArray() (resp.Resp, error) {
	length, err := p.parseInt()
	if err != nil {
		return resp.Resp{}, err
	}
	elements := make([]*resp.Resp, length.Integer)
	for i := range length.Integer {
		value, err := p.ReadResp()
		if err != nil {
			fmt.Printf("Error reading RESP: %v", err)
			return resp.Resp{}, err
		}
		elements[i] = &value
	}
	return resp.Resp{T: resp.Array, Elements: elements}, nil
}

func (p *Parser) parseError() (resp.Resp, error) {
	p.ReadLine()
	return resp.Resp{T: resp.Error}, nil
}

func (p *Parser) parseString() (resp.Resp, error) {
	line, err := p.ReadLine()
	if err != nil {
		return resp.Resp{}, err
	}
	return resp.Resp{T: resp.String, Data: line}, nil
}
