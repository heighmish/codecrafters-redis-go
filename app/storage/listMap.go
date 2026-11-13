package storage

import (
	"fmt"
	"redis-go/app/resp"
	"slices"
	"sync"
)

type ListMap struct {
	Mu        sync.Mutex
	lists     map[string][]resp.Resp
	listeners map[string][]Listener
}

type Listener struct {
	Channel chan byte
	Id      int
}

func NewListMap() *ListMap {
	return &ListMap{
		lists:     make(map[string][]resp.Resp),
		listeners: make(map[string][]Listener),
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

	channels, channelsExist := lm.listeners[key]
	for index, value := range values {
		arr = append(arr, *value)
		if channelsExist && index < len(channels) {
			channels[index].Channel <- 0
		}
	}

	lm.lists[key] = arr
	return len(arr)
}

func (lm *ListMap) Prepend(key string, values []*resp.Resp) int {
	// Horribly inefficient fix it later by building and merging the slices with preallocations
	lm.Mu.Lock()
	defer lm.Mu.Unlock()

	arr, ok := lm.lists[key]
	if !ok {
		fmt.Println("Creating list with key: ", key)
		arr = make([]resp.Resp, 0)
	}

	channels, channelsExist := lm.listeners[key]
	for index, value := range values {
		arr = slices.Insert(arr, 0, *value)
		if channelsExist && index < len(channels) {
			channels[index].Channel <- 0
		}
	}

	lm.lists[key] = arr
	return len(arr)
}

func (lm *ListMap) PopElements(key string, removalCount int) ([]resp.Resp, bool) {
	lm.Mu.Lock()
	defer lm.Mu.Unlock()
	return lm.popElementsLocked(key, removalCount)
}

func (lm *ListMap) GetList(key string) ([]resp.Resp, bool) {
	lm.Mu.Lock()
	defer lm.Mu.Unlock()
	return lm.getListLocked(key)
}

func (lm *ListMap) Blpop(key string, waiterId int) (resp.Resp, bool) {
	lm.Mu.Lock()
	element, ok := lm.popElementsLocked(key, 1)
	if ok {
		lm.Mu.Unlock()
		return element[0], true
	}

	c := make(chan byte)
	// Todo - track goroutines set of listeners and clean up when they leave
	// Todo - i guess add an object tracking thread local state into commandDispatcher state
	listener := Listener{
		Channel: c,
		Id:      waiterId,
	}
	listeners, ok := lm.listeners[key]
	if !ok {
		arr := make([]Listener, 1)
		arr[0] = listener
		lm.listeners[key] = arr
	} else {
		lm.listeners[key] = append(listeners, listener)
	}

	// Channel created, unlock and wait for new client to add an element
	lm.Mu.Unlock()
	_ = <-c
	lm.Mu.Lock()
	listeners, ok = lm.listeners[key]
	if ok {
		lm.listeners[key] = slices.DeleteFunc(listeners, func(listener Listener) bool { return listener.Id == waiterId })
	}

	element, ok = lm.popElementsLocked(key, 1)
	if !ok {
		fmt.Printf("[ERROR BLPOP] - Channel with id %v awoken but no value ready\r\n", waiterId)
		return resp.Resp{}, false
	}
	lm.Mu.Unlock()
	return element[0], true
}

func (lm *ListMap) popElementsLocked(key string, removalCount int) ([]resp.Resp, bool) {
	arr, ok := lm.getListLocked(key)
	if !ok {
		return nil, false
	}

	if len(arr) == 0 {
		delete(lm.lists, key)
		return nil, false
	}

	if removalCount > len(arr) {
		removalCount = len(arr)
	}

	elements := make([]resp.Resp, removalCount)
	copy(elements, arr[0:removalCount])
	lm.lists[key] = slices.Delete(arr, 0, removalCount)
	return elements, true
}

func (lm *ListMap) getListLocked(key string) ([]resp.Resp, bool) {
	val, ok := lm.lists[key]
	return val, ok
}
