package dataflash

import (
	"bytes"
	"io"
	"testing"
)

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
