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
	if err := parser.SetFilter([]string{"GPS"}); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}

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

func TestSetFilterInvalid(t *testing.T) {
	parser, err := NewParser("log_29_2025-12-13-16-08-22.bin")
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	// Try to set filter with invalid names
	err = parser.SetFilter([]string{"INVALID", "NOTEXIST"})
	if err == nil {
		t.Fatal("expected error for invalid filter names, got nil")
	}

	// Mix of valid and invalid should work (at least one valid)
	err = parser.SetFilter([]string{"GPS", "INVALID"})
	if err != nil {
		t.Fatalf("expected no error for mixed filter (at least one valid), got: %v", err)
	}

	// Verify we can still read GPS messages
	msg, err := parser.ReadMessage()
	if err != nil {
		t.Fatalf("error reading message after mixed filter: %v", err)
	}
	if msg.Name != "GPS" {
		t.Errorf("expected GPS, got %s", msg.Name)
	}
}

func TestFilterChangeRewinds(t *testing.T) {
	parser, err := NewParser("log_29_2025-12-13-16-08-22.bin")
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	// Read 5 GPS messages
	if err := parser.SetFilter([]string{"GPS"}); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := parser.ReadMessage(); err != nil {
			t.Fatalf("error reading GPS: %v", err)
		}
	}

	// Change filter to IMU - should rewind automatically
	if err := parser.SetFilter([]string{"IMU"}); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}

	// Should be able to read IMU messages from the beginning
	msg, err := parser.ReadMessage()
	if err != nil {
		t.Fatalf("error reading IMU: %v", err)
	}
	if msg.Name != "IMU" {
		t.Errorf("expected IMU, got %s", msg.Name)
	}
}
