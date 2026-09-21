package dataflash

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
	"testing"
)

func openTestParser(t *testing.T) *Parser {
	t.Helper()
	f, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("failed to open %s: %v", testFile, err)
	}
	t.Cleanup(func() { f.Close() })
	p, err := NewParser(f)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	return p
}

// sameValue compares decoded values by type and printed form, so NaN floats
// (which never compare equal with ==) still match.
func sameValue(a, b any) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(b) && fmt.Sprint(a) == fmt.Sprint(b)
}

func sameFields(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || !sameValue(av, bv) {
			return false
		}
	}
	return true
}

// Get and Fields must return exactly what the eager decoder returns, for every
// message in a real log.
func TestLazyMatchesEagerDecode(t *testing.T) {
	p := openTestParser(t)

	messages := 0
	for {
		msg, err := p.ReadMessage()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		messages++

		want, err := DecodeMessageBody(msg.body, msg.schema)
		if err != nil {
			t.Fatalf("message %d (%s): eager decode failed: %v", msg.LineNo, msg.Name, err)
		}

		for col, wv := range want {
			gv, ok := msg.Get(col)
			if !ok {
				t.Fatalf("message %d (%s): Get(%q) not found", msg.LineNo, msg.Name, col)
			}
			if !sameValue(gv, wv) {
				t.Fatalf("message %d (%s): Get(%q) = %v (%T), want %v (%T)",
					msg.LineNo, msg.Name, col, gv, gv, wv, wv)
			}
		}
		if !sameFields(msg.Fields(), want) {
			t.Fatalf("message %d (%s): Fields() differs from DecodeMessageBody", msg.LineNo, msg.Name)
		}
	}
	if messages == 0 {
		t.Fatal("no messages read")
	}
}

// ReadInto must produce the same sequence as ReadMessage, including when one
// Message (with cached fields) is reused across different message types.
func TestReadIntoMatchesReadMessage(t *testing.T) {
	ref := openTestParser(t)
	reuse := openTestParser(t)

	var msg Message
	for n := 0; ; n++ {
		want, wantErr := ref.ReadMessage()
		gotErr := reuse.ReadInto(&msg)
		if wantErr != gotErr {
			t.Fatalf("message %d: error mismatch: ReadMessage=%v ReadInto=%v", n, wantErr, gotErr)
		}
		if wantErr == io.EOF {
			return
		}
		if wantErr != nil {
			t.Fatalf("message %d: %v", n, wantErr)
		}

		if msg.Type != want.Type || msg.Name != want.Name || msg.LineNo != want.LineNo || msg.TimeUS != want.TimeUS {
			t.Fatalf("message %d: header mismatch: got {%d %s %d %d}, want {%d %s %d %d}", n,
				msg.Type, msg.Name, msg.LineNo, msg.TimeUS,
				want.Type, want.Name, want.LineNo, want.TimeUS)
		}
		if !sameFields(msg.Fields(), want.Fields()) {
			t.Fatalf("message %d (%s): Fields differ between ReadInto and ReadMessage", n, msg.Name)
		}
	}
}

// Values decoded from a reused message must survive the next ReadInto.
func TestReadIntoDecodedValuesAreIndependent(t *testing.T) {
	ref := openTestParser(t)
	reuse := openTestParser(t)

	type snapshot struct {
		fields map[string]any
		want   map[string]any
	}
	var kept []snapshot

	var msg Message
	for i := 0; i < 500; i++ {
		if err := reuse.ReadInto(&msg); err != nil {
			t.Fatalf("ReadInto: %v", err)
		}
		want, err := ref.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		// Keep the cached map from this message across later reads.
		kept = append(kept, snapshot{fields: msg.Fields(), want: want.Fields()})
	}

	for i, s := range kept {
		if !sameFields(s.fields, s.want) {
			t.Fatalf("kept Fields() map of message %d changed after later ReadInto calls", i)
		}
	}
}

// Messages from ReadMessage own their body: reading further must not change them.
func TestReadMessageRetainedMessagesUnchanged(t *testing.T) {
	ref := openTestParser(t)
	p := openTestParser(t)

	var kept []*Message
	var want []map[string]any
	for i := 0; i < 500; i++ {
		msg, err := p.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		kept = append(kept, msg)
		r, err := ref.ReadMessage()
		if err != nil {
			t.Fatalf("reference ReadMessage: %v", err)
		}
		want = append(want, r.Fields())
	}

	for i, msg := range kept {
		if !sameFields(msg.Fields(), want[i]) {
			t.Fatalf("retained message %d (%s) changed after later reads", i, msg.Name)
		}
	}
}

func TestGetEdgeCases(t *testing.T) {
	schema := &Schema{Format: "BH", Columns: "A,B", Length: 6}
	schema.ensureLayout()

	t.Run("missing field", func(t *testing.T) {
		m := &Message{schema: schema, body: []byte{1, 2, 0}}
		if v, ok := m.Get("Nope"); ok || v != nil {
			t.Errorf("Get(Nope) = %v, %v; want nil, false", v, ok)
		}
	})

	t.Run("zero message", func(t *testing.T) {
		var m Message
		if v, ok := m.Get("A"); ok || v != nil {
			t.Errorf("Get on zero Message = %v, %v; want nil, false", v, ok)
		}
		if f := m.Fields(); f == nil || len(f) != 0 {
			t.Errorf("Fields on zero Message = %v; want empty non-nil map", f)
		}
	})

	t.Run("truncated body", func(t *testing.T) {
		m := &Message{schema: schema, body: []byte{7, 0xE8}} // B fits, H does not
		if v, ok := m.Get("A"); !ok || v != uint8(7) {
			t.Errorf("Get(A) = %v, %v; want 7, true", v, ok)
		}
		if v, ok := m.Get("B"); ok || v != nil {
			t.Errorf("Get(B) on truncated body = %v, %v; want nil, false", v, ok)
		}
	})

	t.Run("unknown format char", func(t *testing.T) {
		s := &Schema{Format: "B?", Columns: "A,X", Length: 5}
		s.ensureLayout()
		m := &Message{schema: s, body: []byte{1, 2}}
		if v, ok := m.Get("X"); ok || v != nil {
			t.Errorf("Get(X) = %v, %v; want nil, false", v, ok)
		}
	})
}

func TestDecodeMessageBodyTruncated(t *testing.T) {
	schema := &Schema{Format: "BHI", Columns: "A,B,C", Length: 10}

	got, err := DecodeMessageBody([]byte{9, 0x10, 0x27, 0x01}, schema) // I needs 4 bytes, only 1 left
	if err == nil {
		t.Fatal("expected an error for a truncated body, got nil")
	}
	want := map[string]any{"A": uint8(9), "B": uint16(10000)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("partial result = %v, want %v", got, want)
	}
}

// A schema whose format needs more bytes than the message body must not panic.
func TestFieldsTruncatedBodyDoesNotPanic(t *testing.T) {
	schema := &Schema{Format: "BQ", Columns: "A,B", Length: 6}
	schema.ensureLayout()
	m := &Message{schema: schema, body: []byte{5, 1, 2}}

	got := m.Fields()
	if len(got) != 1 || got["A"] != uint8(5) {
		t.Errorf("Fields() = %v, want only A=5", got)
	}
}

// TimeUS is only extracted from 64-bit formats, as before lazy decoding.
func TestDecodeTimeUS(t *testing.T) {
	tests := []struct {
		name   string
		format string
		body   []byte
		want   int64
	}{
		{"uint64", "QB", []byte{0x10, 0x27, 0, 0, 0, 0, 0, 0, 1}, 10000},
		{"int64", "qB", []byte{0x10, 0x27, 0, 0, 0, 0, 0, 0, 1}, 10000},
		{"uint32 not recognised", "IB", []byte{0x10, 0x27, 0, 0, 1}, 0},
		{"truncated body", "QB", []byte{0x10, 0x27}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &Schema{Format: tc.format, Columns: "TimeUS,X"}
			s.ensureLayout()
			m := &Message{schema: s, body: tc.body}
			m.decodeTimeUS()
			if m.TimeUS != tc.want {
				t.Errorf("TimeUS = %d, want %d", m.TimeUS, tc.want)
			}
		})
	}
}

// Messages from one parser share schemas; concurrent Get on different messages
// must be race-free (run with -race).
func TestConcurrentGetOnDistinctMessages(t *testing.T) {
	p := openTestParser(t)

	var msgs []*Message
	for len(msgs) < 5000 {
		msg, err := p.ReadMessage()
		if err != nil {
			break
		}
		msgs = append(msgs, msg)
	}

	const workers = 4
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each worker owns every workers-th message; only the schemas are shared.
			for i := w; i < len(msgs); i += workers {
				msgs[i].Get("TimeUS")
				msgs[i].Fields()
			}
		}()
	}
	wg.Wait()
}
