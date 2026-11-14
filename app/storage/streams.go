package storage

import (
	"cmp"
	"errors"
	"fmt"
	"redis-go/app/resp"
	"strconv"
	"strings"
	"sync"
)

type StreamMap struct {
	streams map[string]Stream
	mu      sync.Mutex
}

func NewStreamMap() *StreamMap {
	return &StreamMap{streams: make(map[string]Stream)}
}

func (sm *StreamMap) AddEntry(key string, id string, kvs []*resp.Resp) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	entryId, err := NewEntryId(id)
	if err != nil {
		return "", err
	}

	stream, ok := sm.streams[key]
	// checking length should be redundant since we cant have an empty stream (or can we?)
	if ok && len(stream) > 0 {
		lastEntry := stream[len(stream)-1].Id
		if !entryId.IsAfter(lastEntry) {
			return "", errors.New("New entry must be strictly more recent")
		}
	}

	if len(kvs)%2 != 0 {
		return "", fmt.Errorf("Invalid number of entry KV pairs: %v", len(kvs))
	}

	entry := Entry{
		Id:    entryId,
		Pairs: make(map[string]string, len(kvs)/2),
	}
	for i := 0; i < len(kvs); i += 2 {
		entry.Pairs[kvs[i].Data] = kvs[i+1].Data
	}

	if !ok {
		stream = make([]Entry, 1)
		stream[0] = entry
	} else {
		stream = append(stream, entry)
	}

	sm.streams[key] = stream
	return entryId.String(), nil
}

func (sm *StreamMap) GetStream(key string) (Stream, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	stream, ok := sm.streams[key]
	return stream, ok
}

type Stream []Entry

type Entry struct {
	Id    EntryId
	Pairs map[string]string
}

type EntryId struct {
	time           int64
	sequenceNumber int64
}

func (id EntryId) String() string {
	return fmt.Sprintf("%v-%v", id.time, id.sequenceNumber)
}

func (id EntryId) Compare(other EntryId) int {
	return cmp.Or(cmp.Compare(id.time, other.time), cmp.Compare(id.sequenceNumber, other.sequenceNumber))
}

func (id EntryId) IsAfter(other EntryId) bool {
	return id.Compare(other) > 0
}

func NewEntryId(id string) (EntryId, error) {
	time, sequenceNumber, found := strings.Cut(id, "-")
	if !found {
		return EntryId{}, fmt.Errorf("Invalid entryId %v", id)
	}

	timeParsed, err := strconv.ParseInt(time, 10, 64)
	if err != nil {
		return EntryId{}, fmt.Errorf("Invalid entryId %v", id)
	}

	sequenceNumberParsed, err := strconv.ParseInt(sequenceNumber, 10, 64)
	if err != nil {
		return EntryId{}, fmt.Errorf("Invalid entryId %v", id)
	}

	return EntryId{
		time:           timeParsed,
		sequenceNumber: sequenceNumberParsed,
	}, nil
}
