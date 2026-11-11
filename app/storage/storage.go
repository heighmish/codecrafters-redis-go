package storage

import (
	"redis-go/app/resp"
	"sync"
	"time"
)

type storage map[string]Value

type Container struct {
	Mu      sync.Mutex
	Storage storage
}

func NewContainer() Container {
	return Container{
		Storage: make(storage),
	}
}

func (c *Container) Add(key string, value Value) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Storage[key] = value
}

func (c *Container) Get(key string) (Value, bool) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	val, ok := c.Storage[key]
	return val, ok
}

func (c *Container) Delete(key string) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	delete(c.Storage, key)
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
