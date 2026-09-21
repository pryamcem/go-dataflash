package dataflash

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"
)

// fmtRecord builds a FMT (type 128) record defining a schema.
// Layout: HEAD1, HEAD2, 128, Type, Length, Name[4], Format[16], Columns[64].
func fmtRecord(defType, length uint8, name, format, columns string) []byte {
	rec := []byte{HEAD1, HEAD2, FMTType, defType, length}
	for _, f := range []struct {
		s string
		n int
	}{{name, 4}, {format, 16}, {columns, 64}} {
		field := make([]byte, f.n)
		copy(field, f.s)
		rec = append(rec, field...)
	}
	return rec
}

// Regression: a known non-FMT body containing HEAD1/HEAD2 magic must not desync
// buildSchemas and drop the FMT records that follow it.
func TestBuildSchemasMagicInBody(t *testing.T) {
	const (
		fooType  = 10
		baroType = 11
	)

	// FOO body opens with magic + the FMT type byte (0x80): a byte-scan
	// re-aligns here and eats the BARO definition that follows.
	fooBody := make([]byte, 20)
	fooBody[0] = HEAD1
	fooBody[1] = HEAD2
	fooBody[2] = FMTType
	fooLen := uint8(HeaderSize + len(fooBody))

	var log []byte
	log = append(log, fmtRecord(fooType, fooLen, "FOO", "f", "Val")...)
	log = append(log, HEAD1, HEAD2, fooType)
	log = append(log, fooBody...)
	log = append(log, fmtRecord(baroType, 11, "BARO", "If", "TimeUS,Alt")...)

	parser, err := NewParser(bytes.NewReader(log))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	schemas := parser.GetSchemas()

	if foo, ok := schemas[fooType]; !ok || foo.Name != "FOO" {
		t.Fatalf("FOO schema missing or wrong: %+v", schemas[fooType])
	}

	baro, ok := schemas[baroType]
	if !ok {
		t.Fatal("BARO schema was dropped: parser desynced on magic bytes in FOO body")
	}
	if baro.Name != "BARO" {
		t.Errorf("expected BARO, got %q", baro.Name)
	}
}

var errBoom = errors.New("boom")

// failingSource is a seekable source that returns failAtEOF, when set, instead
// of io.EOF once its data runs out, like a read error at the end of the data.
type failingSource struct {
	*bytes.Reader
	failAtEOF error
}

func (s *failingSource) Read(p []byte) (int, error) {
	n, err := s.Reader.Read(p)
	if n == 0 && s.failAtEOF != nil {
		return 0, s.failAtEOF
	}
	return n, err
}

// returnsWithin runs f and fails the test if it does not return in time, so a
// hang shows up as a failure instead of a stuck test run.
func returnsWithin(t *testing.T, f func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- f() }()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("did not return within 2s (stuck in a loop?)")
		return nil
	}
}

func robustnessLog(tail ...byte) []byte {
	log := fmtRecord(FMTType, FMTLength, "FMT", "BBnNZ", "Type,Length,Name,Format,Columns")
	log = append(log, fmtRecord(10, 4, "AAAA", "B", "V")...)
	log = append(log, HEAD1, HEAD2, 10, 1)
	return append(log, tail...)
}

// A read error that is not EOF must be returned, not retried forever.
func TestReadReturnsSourceError(t *testing.T) {
	tails := map[string][]byte{
		"error after last message": nil,
		"error while resyncing":    {1, 2, 3, 4, 5}, // junk with no header after it
	}
	readers := map[string]func(p *Parser) error{
		"ReadMessage": func(p *Parser) error { _, err := p.ReadMessage(); return err },
		"ReadInto":    func(p *Parser) error { var m Message; return p.ReadInto(&m) },
	}

	for tailName, tail := range tails {
		for readerName, read := range readers {
			t.Run(tailName+"/"+readerName, func(t *testing.T) {
				src := &failingSource{Reader: bytes.NewReader(robustnessLog(tail...))}
				p, err := NewParser(src)
				if err != nil {
					t.Fatalf("NewParser: %v", err)
				}
				src.failAtEOF = errBoom

				err = returnsWithin(t, func() error {
					for {
						if err := read(p); err != nil {
							return err
						}
					}
				})
				if !errors.Is(err, errBoom) {
					t.Errorf("got error %v, want %v", err, errBoom)
				}
			})
		}
	}
}

// A read error while building schemas must fail NewParser, not hang it.
func TestNewParserReturnsSourceError(t *testing.T) {
	src := &failingSource{Reader: bytes.NewReader(robustnessLog()), failAtEOF: errBoom}

	err := returnsWithin(t, func() error {
		_, err := NewParser(src)
		return err
	})
	if !errors.Is(err, errBoom) {
		t.Errorf("NewParser error = %v, want %v", err, errBoom)
	}
}

// Junk in front of a FMT record must not make schema building miss it.
func TestBuildSchemasResyncsAfterJunk(t *testing.T) {
	log := fmtRecord(FMTType, FMTLength, "FMT", "BBnNZ", "Type,Length,Name,Format,Columns")
	log = append(log, 1, 2, 3, 4, 5) // 5 bytes: not a multiple of the 3-byte header
	log = append(log, fmtRecord(10, 4, "AAAA", "B", "V")...)

	p, err := NewParser(bytes.NewReader(log))
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	if s, ok := p.GetSchemas()[10]; !ok || s.Name != "AAAA" {
		t.Errorf("schema for type 10 not found after junk, got %+v", p.GetSchemas())
	}
}

func TestStats(t *testing.T) {
	good := []byte{HEAD1, HEAD2, 10, 1}

	tests := []struct {
		name string
		junk []byte // inserted between two good messages
		want Stats
	}{
		{"clean", nil, Stats{}},
		// 3-byte bad header, then 2 more bytes skipped looking for the next header
		{"junk bytes", []byte{1, 2, 3, 4, 5}, Stats{SkippedBytes: 5, Resyncs: 1, InvalidHeaders: 1}},
		// unknown type 99: its 3-byte header, then 2 junk bytes
		{"unknown type", []byte{HEAD1, HEAD2, 99, 1, 2}, Stats{SkippedBytes: 5, Resyncs: 1, UnknownTypes: 1}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log := fmtRecord(FMTType, FMTLength, "FMT", "BBnNZ", "Type,Length,Name,Format,Columns")
			log = append(log, fmtRecord(10, 4, "AAAA", "B", "V")...)
			log = append(log, good...)
			log = append(log, tc.junk...)
			log = append(log, good...)

			p, err := NewParser(bytes.NewReader(log))
			if err != nil {
				t.Fatalf("NewParser: %v", err)
			}
			readAll := func() {
				for {
					_, err := p.ReadMessage()
					if err == io.EOF {
						return
					}
					if err != nil {
						t.Fatalf("ReadMessage: %v", err)
					}
				}
			}

			readAll()
			if got := p.Stats(); got != tc.want {
				t.Errorf("Stats = %+v, want %+v", got, tc.want)
			}

			// Rewind starts a new pass: zero at first, then the same damage again.
			if err := p.Rewind(); err != nil {
				t.Fatalf("Rewind: %v", err)
			}
			if got := p.Stats(); got != (Stats{}) {
				t.Errorf("Stats after Rewind = %+v, want zero", got)
			}
			readAll()
			if got := p.Stats(); got != tc.want {
				t.Errorf("Stats on second pass = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// However a log ends, reading finishes with plain io.EOF; only Stats tells a
// clean end from a log cut off in the middle of a message.
func TestReadEndsWithEOF(t *testing.T) {
	tests := []struct {
		name      string
		tail      []byte
		filter    string
		truncated bool
	}{
		{"clean end", nil, "", false},
		{"cut inside header", []byte{HEAD1, HEAD2}, "", true},
		{"cut inside body", []byte{HEAD1, HEAD2, 11, 1, 2}, "", true},
		{"cut inside a filtered-out body", []byte{HEAD1, HEAD2, 10, 1}, "CCCC", true},
	}
	readers := map[string]func(p *Parser) error{
		"ReadMessage": func(p *Parser) error { _, err := p.ReadMessage(); return err },
		"ReadInto":    func(p *Parser) error { var m Message; return p.ReadInto(&m) },
	}

	for _, tc := range tests {
		for name, read := range readers {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				// types 10 (AAAA) and 11 (CCCC) both have a 3-byte body
				log := fmtRecord(FMTType, FMTLength, "FMT", "BBnNZ", "Type,Length,Name,Format,Columns")
				log = append(log, fmtRecord(10, 6, "AAAA", "BBB", "X,Y,Z")...)
				log = append(log, fmtRecord(11, 6, "CCCC", "BBB", "X,Y,Z")...)
				log = append(log, HEAD1, HEAD2, 11, 1, 2, 3)
				log = append(log, tc.tail...)

				p, err := NewParser(bytes.NewReader(log))
				if err != nil {
					t.Fatalf("NewParser: %v", err)
				}
				if tc.filter != "" {
					if err := p.SetFilter(tc.filter); err != nil {
						t.Fatalf("SetFilter: %v", err)
					}
				}

				var last error
				for last = nil; last == nil; last = read(p) {
				}
				if last != io.EOF {
					t.Errorf("reading ended with %v, want io.EOF", last)
				}
				if got := p.Stats().Truncated; got != tc.truncated {
					t.Errorf("Stats().Truncated = %v, want %v", got, tc.truncated)
				}
			})
		}
	}
}
