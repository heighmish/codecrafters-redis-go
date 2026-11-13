package storage

import (
	"fmt"
	"sync"
	"time"
)

type storage map[string]Value

type Container struct {
	Mu      sync.Mutex // Replace with RWMutex
	Storage storage
}

func NewContainer() *Container {
	return &Container{
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
	if val.ExpiryTime != nil {
		if time.Now().UTC().Compare(*val.ExpiryTime) > 0 {
			fmt.Printf("Key %s expired\r\n", key)
			delete(c.Storage, key)
			return Value{}, false
		}
	}
	return val, ok
}
