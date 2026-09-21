package dataflash

import (
	"bytes"
	"io"
	"testing"
)

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
					if err == io.EOF || err == io.ErrUnexpectedEOF {
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
