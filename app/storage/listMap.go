package storage

import (
	"fmt"
	"redis-go/app/resp"
	"slices"
	"sync"
	"time"
)

type ListMap struct {
	Mu        sync.Mutex
	lists     map[string][]resp.Resp
	listeners map[string][]Listener
}

type Listener struct {
	Channel chan resp.Resp
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
	index := 0
	channelsPublished := 0
	channels, ok := lm.listeners[key]
	if ok {
		for ; index < len(values) && index < len(channels); index++ {
			channels[index].Channel <- *values[index]
		}
		channelsPublished = index
		lm.listeners[key] = slices.Delete(channels, 0, index)
	}

	arr, ok := lm.lists[key]
	if !ok {
		fmt.Println("Creating list with key: ", key)
		arr = make([]resp.Resp, 0)
	}

	for ; index < len(values); index++ {
		arr = append(arr, *values[index])
	}

	lm.lists[key] = arr
	return len(arr) + channelsPublished
}

func (lm *ListMap) Prepend(key string, values []*resp.Resp) int {
	// Horribly inefficient fix it later by building and merging the slices with preallocations
	lm.Mu.Lock()
	defer lm.Mu.Unlock()
	index := 0
	channelsPublished := 0
	channels, ok := lm.listeners[key]
	if ok {
		for ; index < len(values) && index < len(channels); index++ {
			channels[index].Channel <- *values[index]
		}
		channelsPublished = index
		lm.listeners[key] = slices.Delete(channels, 0, index)
	}

	arr, ok := lm.lists[key]
	if !ok {
		fmt.Println("Creating list with key: ", key)
		arr = make([]resp.Resp, 0)
	}

	for ; index < len(values); index++ {
		arr = slices.Insert(arr, 0, *values[index])
	}

	lm.lists[key] = arr
	return len(arr) + channelsPublished
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

func (lm *ListMap) Blpop(key string, timeout float64, waiterId int) (resp.Resp, bool) {
	lm.Mu.Lock()
	element, ok := lm.popElementsLocked(key, 1)
	if ok {
		lm.Mu.Unlock()
		return element[0], true
	}

	c := make(chan resp.Resp, 1)
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
	if timeout != 0 {
		block := time.After(time.Duration(timeout * float64(time.Second)))
		select {
		case v := <-c:
			return v, true
		case <-block:
			// Todo!
			// Introduce some extra state on the listener to ensure race condition doesnt happen where timeout occurs as push pushes in
			// Add state - WAITING, DELIVERED, CANCELLED, TIMEOUT
			lm.CleanUpListeners(key, waiterId)
			return resp.Resp{}, false
		}
	}
	v := <-c
	return v, true
}

func (lm *ListMap) CleanUpListeners(key string, waiterId int) {
	lm.Mu.Lock()
	defer lm.Mu.Unlock()
	listeners, ok := lm.listeners[key]
	if !ok {
		return
	}

	lm.listeners[key] = slices.DeleteFunc(listeners, func(l Listener) bool { return l.Id == waiterId })
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
