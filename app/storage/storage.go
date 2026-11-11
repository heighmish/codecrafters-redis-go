package storage

import (
	"fmt"
	"redis-go/app/resp"
	"sync"
	"time"
)

type storage map[string]Value

type ListMap struct {
	Mu    sync.Mutex
	lists map[string][]resp.Resp
}

func NewListMap() *ListMap {
	return &ListMap{
		lists: make(map[string][]resp.Resp),
	}
}

func (lm *ListMap) Push(key string, value resp.Resp) int {
	lm.Mu.Lock()
	defer lm.Mu.Unlock()
	arr, ok := lm.lists[key]
	if !ok {
		fmt.Println("Creating list with key: ", key)
		newArr := make([]resp.Resp, 1)
		newArr[0] = value
		lm.lists[key] = newArr
		return 1
	}
	arr = append(arr, value)
	return len(arr)
}

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
