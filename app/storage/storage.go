package storage

import (
	"redis-go/app/resp"
	"redis-go/app/storage/streams"
	"time"
)

type Cache struct {
	Lists     *ListMap
	Container *Container
	Streams   *streams.StreamMap
}

func NewCache() *Cache {
	return &Cache{
		Lists:     NewListMap(),
		Container: NewContainer(),
		Streams:   streams.NewStreamMap(),
	}
}

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
