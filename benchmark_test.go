package dataflash

import (
	"io"
	"os"
	"testing"
)

var benchmarkFile = os.Getenv("DATAFLASH_BENCH_FILE")

func BenchmarkParseAllMessages(b *testing.B) {
	if benchmarkFile == "" {
		b.Skip("set DATAFLASH_BENCH_FILE to run benchmarks")
	}
	for b.Loop() {
		f, err := os.Open(benchmarkFile)
		if err != nil {
			b.Fatalf("failed to open file: %v", err)
		}
		parser, err := NewParser(f)
		if err != nil {
			f.Close()
			b.Fatalf("failed to create parser: %v", err)
		}

		for {
			_, err := parser.ReadMessage()
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				f.Close()
				b.Fatalf("error reading message: %v", err)
			}
		}
		f.Close()
	}
}

func BenchmarkParseFiltered(b *testing.B) {
	if benchmarkFile == "" {
		b.Skip("set DATAFLASH_BENCH_FILE to run benchmarks")
	}
	for b.Loop() {
		f, err := os.Open(benchmarkFile)
		if err != nil {
			b.Fatalf("failed to open file: %v", err)
		}
		parser, err := NewParser(f)
		if err != nil {
			f.Close()
			b.Fatalf("failed to create parser: %v", err)
		}

		if err := parser.SetFilter("GPS", "IMU"); err != nil {
			f.Close()
			b.Fatalf("failed to set filter: %v", err)
		}

		for {
			_, err := parser.ReadMessage()
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				f.Close()
				b.Fatalf("error reading message: %v", err)
			}
		}
		f.Close()
	}
}
