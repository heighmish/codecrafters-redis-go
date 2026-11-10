package storage

import (
	"redis-go/app/resp"
	"time"
)

type Storage map[string]Value

type Value struct {
	Data       string
	ExpiryTime *time.Time
}

func NewValue(r resp.Resp) Value {
	return Value{
		Data:       r.Data,
		ExpiryTime: nil,
	}
}

func NewValueWithExpiry(r resp.Resp, t *time.Time) Value {
	return Value{
		Data:       r.Data,
		ExpiryTime: t,
	}
}
