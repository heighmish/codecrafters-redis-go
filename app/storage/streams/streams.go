package streams

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"redis-go/app/resp"
	"redis-go/app/storage/listener"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type idMode int

const (
	idInvalid      idMode = iota
	idExplicit            // "5-3"
	idAutoSequence        // "5-*"
	idAuto                // "*"
)

var (
	ErrInvalidId  = errors.New("Invalid stream ID specified as stream command argument")
	ErrZeroId     = errors.New("The ID specified in XADD must be greater than 0-0")
	ErrIdTooSmall = errors.New("The ID specified in XADD is equal or smaller than the target stream top item")
)

type TimeoutError struct{}

func (e *TimeoutError) Error() string {
	return "Timeout Error Occured"
}

type RangeRequest struct {
	Key   string
	Start string
}

var (
	MinEntryId = EntryId{time: 0, sequenceNumber: 0}
	MaxEntryId = EntryId{time: math.MaxInt64, sequenceNumber: math.MaxInt64}
)

type entryIdRequest struct {
	mode           idMode
	time           int64
	sequenceNumber int64
}

func (s Stream) nextId(raw string, now int64) (EntryId, error) {
	req, err := parseEntryIdRequest(raw)
	if err != nil {
		return EntryId{}, err
	}
	lastId := EntryId{time: 0, sequenceNumber: 0}
	if len(s.Entries) > 0 {
		lastId = s.Entries[len(s.Entries)-1].Id
	}
	id := resolveEntryIdRequest(req, lastId, now)
	if err := validate(id, lastId); err != nil {
		return EntryId{}, err
	}
	return id, nil
}

func parseEntryIdRequest(s string) (entryIdRequest, error) {
	if s == "*" {
		return entryIdRequest{mode: idAuto}, nil
	}

	time, seq, found := strings.Cut(s, "-")
	if !found {
		return entryIdRequest{}, ErrInvalidId
	}

	timeParsed, err := strconv.ParseInt(time, 10, 64)
	if err != nil {
		return entryIdRequest{}, ErrInvalidId
	}

	if seq == "*" {
		return entryIdRequest{mode: idAutoSequence, time: timeParsed}, nil
	}

	sequenceParsed, err := strconv.ParseInt(seq, 10, 64)
	if err != nil {
		return entryIdRequest{}, ErrInvalidId
	}

	return entryIdRequest{mode: idExplicit, time: timeParsed, sequenceNumber: sequenceParsed}, nil
}

func resolveEntryIdRequest(req entryIdRequest, last EntryId, now int64) EntryId {
	switch req.mode {
	case idAuto:
		if now > last.time {
			return EntryId{now, 0}
		}
		return EntryId{last.time, last.sequenceNumber + 1}
	case idAutoSequence:
		if req.time == last.time {
			return EntryId{time: req.time, sequenceNumber: last.sequenceNumber + 1}
		}
		return EntryId{time: req.time, sequenceNumber: 0}
	case idExplicit:
		return EntryId{time: req.time, sequenceNumber: req.sequenceNumber}
	}
	panic("[ResolveEntryIdRequest] - unreachable req mode")
}

func validate(id, last EntryId) error {
	if id == (EntryId{}) {
		return ErrZeroId
	}

	if !id.IsAfter(last) {
		return ErrIdTooSmall
	}

	return nil
}

type StreamMap struct {
	streams map[string]Stream
	mu      sync.Mutex
}

func NewStreamMap() *StreamMap {
	return &StreamMap{streams: make(map[string]Stream)}
}

func (sm *StreamMap) AddEntry(key string, id string, kvs []resp.Resp) (string, error) {
	if len(kvs) == 0 || len(kvs)%2 != 0 {
		return "", fmt.Errorf("Invalid number of entry KV pairs: %v", len(kvs))
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	stream := sm.streams[key] // nil slice if does not exist
	entryId, err := stream.nextId(id, time.Now().UnixMilli())
	if err != nil {
		return "", err
	}

	pairs := make([]Pair, 0, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		pairs = append(pairs, Pair{key: kvs[i].Data, value: kvs[i+1].Data})
	}

	stream.Entries = append(stream.Entries, Entry{Id: entryId, Pairs: pairs})
	sm.streams[key] = stream
	for _, l := range sm.streams[key].Listeners.Listeners {
		fmt.Println("Publishing wake up to: ", l.Id)
		l.Channel <- resp.Resp{}
	}
	return entryId.String(), nil
}

func (sm *StreamMap) GetStream(key string) (Stream, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	stream, ok := sm.streams[key]
	return stream, ok
}

func parseEntryIdQuery(id string, start bool) (EntryId, error) {
	if id == "-" {
		return MinEntryId, nil
	}
	if id == "+" {
		return MaxEntryId, nil
	}
	time, sequence, found := strings.Cut(id, "-")
	if !found {
		timeParsed, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return EntryId{}, errors.New("Unable to parse start id")
		}
		if !start {
			return EntryId{time: timeParsed, sequenceNumber: math.MaxInt64}, nil
		}
		return EntryId{time: timeParsed, sequenceNumber: 0}, nil
	}
	timeParsed, err := strconv.ParseInt(time, 10, 64)
	if err != nil {
		return EntryId{}, errors.New("Unable to parse start id")
	}

	sequenceParsed, err := strconv.ParseInt(sequence, 10, 64)
	if err != nil {
		return EntryId{}, errors.New("Unable to parse start id")
	}

	return EntryId{time: timeParsed, sequenceNumber: sequenceParsed}, nil
}

func (sm *StreamMap) GetRange(key string, start string, end string) (Stream, error) {
	startId, err := parseEntryIdQuery(start, true)
	if err != nil {
		return Stream{}, err
	}

	endId, err := parseEntryIdQuery(end, false)
	if err != nil {
		return Stream{}, err
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	stream, ok := sm.streams[key]
	if !ok {
		return Stream{}, nil
	}

	r, err := stream.getRangeLocked(key, startId, endId, false)
	return r, err
}

func (sm *StreamMap) getStreamLocked(key string) Stream {
	stream, ok := sm.streams[key]
	if !ok {
		sm.streams[key] = Stream{}
	}
	return stream
}

func (sm *StreamMap) GetRanges(requests []RangeRequest, blockMilliseconds int64, listenerId int) ([]Stream, error) {
	streams := make([]Stream, 0, len(requests))
	dataRead := false
	timeout := false
	for {
		sm.mu.Lock()
		for i := range requests {
			stream := sm.getStreamLocked(requests[i].Key)

			if requests[i].Start == "$" {
				if len(stream.Entries) > 0 {
					requests[i].Start = stream.Entries[len(stream.Entries)-1].Id.String()
				} else {
					requests[i].Start = MinEntryId.String()
				}
			}
			startId, err := parseEntryIdQuery(requests[i].Start, true)
			if err != nil {
				fmt.Println("Error parsing entryId", requests[i].Start)
				sm.mu.Unlock()
				return streams, err
			}
			arr, err := stream.getRangeLocked(requests[i].Key, startId, MaxEntryId, true)
			if err != nil {
				fmt.Println("Error getting range", requests[i].Key)
				sm.mu.Unlock()
				return streams, err
			}
			if len(arr.Entries) > 0 {
				dataRead = true
				streams = append(streams, arr)
			}
		}

		if blockMilliseconds == -1 {
			// No timeout case
			fmt.Println("No timeout configured - returning")
			sm.mu.Unlock()
			return streams, nil
		}

		if dataRead {
			fmt.Println("Data read in stream - returning")
			sm.mu.Unlock()
			return streams, nil
		}

		c := sm.subscribeLocked(requests, listenerId)
		sm.mu.Unlock()
		if blockMilliseconds == 0 {
			<-c
		} else {
			dur := time.Duration(float64(blockMilliseconds) * float64(time.Millisecond))
			println("Blocking for: ", dur)
			block := time.After(dur)
			select {
			case <-c:
				println("Received value in channel")
				timeout = false
			case <-block:
				println("Timeout waiting for channel")
				timeout = true
			}
		}
		sm.CleanupListeners(streams, listenerId)
		if timeout {
			return streams, &TimeoutError{}
		}
	}
}

func (sm *StreamMap) CleanupListeners(streams []Stream, channelId int) {
	fmt.Println("Cleaning up listeners for channel: ", channelId)
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, s := range streams {
		s.Listeners.Listeners = slices.DeleteFunc(s.Listeners.Listeners, func(l listener.Listener) bool { return l.Id == channelId })
	}
}

func (sm *StreamMap) subscribeLocked(requests []RangeRequest, channelId int) chan resp.Resp {
	channel := make(chan resp.Resp, 1)
	for _, request := range requests {
		fmt.Printf("Registering channel %v on stream %v\r\n", channelId, request.Key)
		stream := sm.getStreamLocked(request.Key)
		stream.Listeners.Listeners = append(stream.Listeners.Listeners, listener.Listener{Channel: channel, Id: channelId})
		sm.streams[request.Key] = stream
	}

	return channel
}

func (s *Stream) getRangeLocked(key string, start, end EntryId, excludeStart bool) (Stream, error) {
	startIdx := s.binarySearch(start, func(target, mid EntryId) bool { return mid.Compare(target) < 0 })
	endIdx := s.binarySearch(end, func(target, mid EntryId) bool { return mid.Compare(target) <= 0 })

	if excludeStart && startIdx < len(s.Entries) && start == s.Entries[startIdx].Id {
		startIdx += 1
	}

	if startIdx >= endIdx {
		return Stream{}, nil
	}

	return Stream{
		Name:    key,
		Entries: slices.Clone(s.Entries[startIdx:endIdx]),
	}, nil

}

func (s *Stream) binarySearch(target EntryId, pred func(target, mid EntryId) bool) int {
	left := -1
	right := len(s.Entries)
	for right-left > 1 {
		mid := left + (right-left)/2
		if pred(target, (s.Entries)[mid].Id) {
			left = mid
		} else {
			right = mid
		}
	}
	return right
}

func EncodeAsResp(streams []Stream) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%v\r\n", len(streams))
	for _, stream := range streams {
		fmt.Fprintf(&sb, "*2\r\n")
		fmt.Fprintf(&sb, "%s", resp.ToBulkString(stream.Name))
		fmt.Fprint(&sb, stream.EncodeAsResp())
	}
	return sb.String()
}

func (s *Stream) EncodeAsResp() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%v\r\n", len(s.Entries))
	for _, entry := range s.Entries {
		fmt.Fprintf(&sb, "*2\r\n")
		fmt.Fprintf(&sb, "%s", resp.ToBulkString(entry.Id.String()))
		fmt.Fprintf(&sb, "*%v\r\n", len(entry.Pairs)*2)
		for _, pair := range entry.Pairs {
			fmt.Fprintf(&sb, "%s", resp.ToBulkString(pair.key))
			fmt.Fprintf(&sb, "%s", resp.ToBulkString(pair.value))
		}
	}

	return sb.String()
}

type Stream struct {
	Name      string
	Entries   []Entry
	Listeners listener.Listeners
}

func NewStream(key string) *Stream {
	return &Stream{
		Name:      key,
		Entries:   make([]Entry, 0),
		Listeners: listener.Listeners{},
	}
}

type Pair struct {
	key   string
	value string
}

type Entry struct {
	Id    EntryId
	Pairs []Pair
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
