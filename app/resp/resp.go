package resp

import (
	"fmt"
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
	T        Type
	Data     string
	Elements []Resp
	Integer  int
}

func (r *Resp) ToBulkString() string {
	if r.Data == "" {
		return "-1\r\n"
	}
	return fmt.Sprintf("$%d\r\n%s\r\n", len(r.Data), r.Data)
}

func (r *Resp) String() string {
	switch r.T {
	case Array:
		var sb strings.Builder
		for _, r := range r.Elements {
			sb.WriteString(r.String())
		}
		return sb.String()
	case Bulk:
		return ToBulkString(r.Data)
	case Error:
		return ToError(r.Data)
	case Integer:
		return ToRespInt(r.Integer)
	case String:
		return ToSimpleString(r.Data)
	}
	fmt.Printf("[Resp To String] Unreachable - %v\r\n", r.T)
	panic("[Resp To String] - unreachable")
}

func ToRespArray(arr []Resp) string {
	len := len(arr)
	var sb strings.Builder
	for _, r := range arr {
		sb.WriteString(r.String())
	}
	return fmt.Sprintf("*%v\r\n%v", len, sb.String())
}

func ToRespInt(i int) string {
	return fmt.Sprintf(":%v\r\n", i)
}

func ToBulkString(s string) string {
	if s == "" {
		return "$-1\r\n"
	}
	return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
}

func ToSimpleString(s string) string {
	return fmt.Sprintf("+%v\r\n", s)
}

func ToError(s string) string {
	return fmt.Sprintf("-ERR %s\r\n", s)
}
