package dataflash

import (
	"io"
	"testing"
)

func TestParserFilter(t *testing.T) {
	// Create parser
	parser, err := NewParser("log_29_2025-12-13-16-08-22.bin")
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	// Set filter to only GPS
	parser.SetFilter([]string{"GPS"})

	// Read 10 messages
	for range 10 {
		msg, err := parser.ReadMessage()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("error reading message: %v", err)
		}

		// Verify it's GPS
		if msg.Name != "GPS" {
			t.Errorf("expected GPS, got %s", msg.Name)
		}
	}
}
