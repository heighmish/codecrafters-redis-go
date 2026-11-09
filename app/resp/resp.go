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

func ToBulkString(s string) string {
	if s == "" {
		return "-1\r\n"
	}
	return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
}
