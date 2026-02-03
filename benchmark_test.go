package dataflash

import (
	"io"
	"testing"
)

const benchmarkFile = "/home/pryamcem/Drones/NORDA/Misc/gps-records/log_23_2026-1-21-17-45-58.bin"

func BenchmarkParseAllMessages(b *testing.B) {
	for b.Loop() {
		parser, err := NewParser(benchmarkFile)
		if err != nil {
			b.Fatalf("failed to create parser: %v", err)
		}

		for {
			_, err := parser.ReadMessage()
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				b.Fatalf("error reading message: %v", err)
			}
		}
		parser.Close()
	}
}

func BenchmarkParseFiltered(b *testing.B) {
	for b.Loop() {
		parser, err := NewParser(benchmarkFile)
		if err != nil {
			b.Fatalf("failed to create parser: %v", err)
		}

		if err := parser.SetFilter("GPS", "IMU"); err != nil {
			b.Fatalf("failed to set filter: %v", err)
		}

		for {
			_, err := parser.ReadMessage()
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				b.Fatalf("error reading message: %v", err)
			}
		}
		parser.Close()
	}
}
