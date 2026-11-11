package storage

import (
	"fmt"
	"redis-go/app/resp"
	"slices"
	"sync"
	"time"
)

type Cache struct {
	Lists     *ListMap
	Container *Container
}

func NewCache() *Cache {
	return &Cache{
		Lists:     NewListMap(),
		Container: NewContainer(),
	}
}

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

func (lm *ListMap) Push(key string, values []*resp.Resp) int {
	lm.Mu.Lock()
	defer lm.Mu.Unlock()
	arr, ok := lm.lists[key]
	if !ok {
		fmt.Println("Creating list with key: ", key)
		arr = make([]resp.Resp, 0)
	}
	for _, val := range values {
		arr = append(arr, *val)
	}
	lm.lists[key] = arr
	return len(arr)
}

func (lm *ListMap) Prepend(key string, values []*resp.Resp) int {
	// Horribly inefficient fix it later
	lm.Mu.Lock()
	defer lm.Mu.Unlock()
	arr, ok := lm.lists[key]
	if !ok {
		fmt.Println("Creating list with key: ", key)
		arr = make([]resp.Resp, 0)
	}
	for _, val := range values {
		arr = slices.Insert(arr, 0, *val)
	}
	lm.lists[key] = arr
	return len(arr)
}

func (lm *ListMap) PopElement(key string) (*resp.Resp, bool) {
	lm.Mu.Lock()
	defer lm.Mu.Unlock()
	arr, ok := lm.lists[key]
	if !ok {
		return nil, false
	}

	if len(arr) == 0 {
		return nil, false
	}

	element := arr[0]
	lm.lists[key] = slices.Delete(arr, 0, 1)
	return &element, true
}

func (lm *ListMap) GetList(key string) ([]resp.Resp, bool) {
	lm.Mu.Lock()
	defer lm.Mu.Unlock()
	val, ok := lm.lists[key]
	return val, ok
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
