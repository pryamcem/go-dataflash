package dataflash

import (
	"io"
	"testing"
)

func TestFilterChangeRewinds(t *testing.T) {
	parser, err := NewParser("log_29_2025-12-13-16-08-22.bin")
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	// Read 5 GPS messages
	parser.SetFilter([]string{"GPS"})
	gpsCount := 0
	for gpsCount < 5 {
		_, err := parser.ReadMessage()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("error reading GPS: %v", err)
		}
		gpsCount++
	}

	if gpsCount != 5 {
		t.Fatalf("expected 5 GPS messages, got %d", gpsCount)
	}

	// Change filter to IMU - should rewind automatically
	parser.SetFilter([]string{"IMU"})
	
	// Should be able to read IMU messages from the beginning
	imuCount := 0
	for imuCount < 5 {
		msg, err := parser.ReadMessage()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("error reading IMU: %v", err)
		}
		if msg.Name != "IMU" {
			t.Errorf("expected IMU, got %s", msg.Name)
		}
		imuCount++
	}

	if imuCount < 5 {
		t.Errorf("expected at least 5 IMU messages, got %d", imuCount)
	}
}
