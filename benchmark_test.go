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
			if err == io.EOF {
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
			if err == io.EOF {
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

// BenchmarkParseReadInto parses every message through the reusable ReadInto
// path, which shares one Message and its body buffer across the whole file.
func BenchmarkParseReadInto(b *testing.B) {
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

		var msg Message
		for {
			err := parser.ReadInto(&msg)
			if err == io.EOF {
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
