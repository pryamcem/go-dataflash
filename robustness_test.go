package dataflash

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

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
