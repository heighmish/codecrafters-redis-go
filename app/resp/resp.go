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
	T      Type
	Data   []byte
	Length int
}

func (r *Resp) ToBulkString() string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(r.Data), r.Data)
}
