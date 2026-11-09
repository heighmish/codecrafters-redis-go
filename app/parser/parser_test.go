package parser

import (
	"redis-go/app/resp"
	"strings"
	"testing"
)

func TestParseBulkString(t *testing.T) {
	expect := "HELLO"
	input := "$5\r\nHELLO\r\n"
	parser := NewParser(strings.NewReader(input))
	got, err := parser.ReadResp()
	if err != nil {
		t.Fatalf("ReadResp() returned an error: %v", err)
	}

	if got.T != resp.Bulk {
		t.Fatalf("got type = %v, want %v ", got.T, resp.Bulk)
	}

	if got.Data != expect {
		t.Fatalf("got value = %v, want %v ", got.Data, expect)
	}
}

func TestParseArray(t *testing.T) {
	input := "*2\r\n*1\r\n$3\r\nboo\r\n:-5\r\n"
	parser := NewParser(strings.NewReader(input))
	got, err := parser.ReadResp()
	if err != nil {
		t.Fatalf("ReadResp() returned an error: %v", err)
	}

	if got.T != resp.Array {
		t.Fatalf("got type = %v, want %v ", got.T, resp.Array)
	}
	if len(got.Elements) != 2 {
		t.Fatalf("Expected array of length %v, got: %v", len(got.Elements), 2)
	}
	subResp := got.Elements[0]
	if subResp.T != resp.Array {
		t.Fatalf("Expected element at index 0 to be subarray")
	}

	if len(subResp.Elements) != 1 {
		t.Fatalf("Expected subarray of length %v, got: %v", len(subResp.Elements), 1)
	}

	if subResp.Elements[0].Data != "boo" {
		t.Fatalf("Expected subarray to contain %v, got %v instead", "boo", subResp.Elements[0])
	}

	if got.Elements[1].Integer != -5 {
		t.Fatalf("Expected second array element to be %v, got %v", -5, got.Elements[1].Integer)
	}
}
