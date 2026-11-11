package resp

import "fmt"

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
	Elements []*Resp
	Integer  int
}

func (r *Resp) ToBulkString() string {
	if r.Data == "" {
		return "-1\r\n"
	}
	return fmt.Sprintf("$%d\r\n%s\r\n", len(r.Data), r.Data)
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
