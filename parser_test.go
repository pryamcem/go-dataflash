package dataflash

import (
	"io"
	"testing"
)

const (
	// Origin: https://discuss.ardupilot.org/t/vtol-crash-after-transition-to-fbwa/138484
	testFile = "testdata/testlog.bin"
)

func TestParserFilter(t *testing.T) {
	// Create parser
	parser, err := NewParser(testFile)
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
	parser, err := NewParser(testFile)
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
	parser, err := NewParser(testFile)
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

func TestMessageTracking(t *testing.T) {
	parser, err := NewParser(testFile)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	// Filter for IMU which should have TimeUS
	if err := parser.SetFilter([]string{"IMU"}); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}

	// Read first message
	msg, err := parser.ReadMessage()
	if err != nil {
		t.Fatalf("error reading message: %v", err)
	}

	// Verify LineNo tracking
	if msg.LineNo == 0 {
		t.Error("expected LineNo to be non-zero")
	}

	// Verify TimeUS extraction
	if msg.TimeUS == 0 {
		t.Error("expected TimeUS to be non-zero for IMU message")
	}

	// Verify TimeUS matches Fields
	if timeUSField, ok := msg.Fields["TimeUS"]; ok {
		var fieldTimeUS int64
		switch v := timeUSField.(type) {
		case int64:
			fieldTimeUS = v
		case uint64:
			fieldTimeUS = int64(v)
		}
		if fieldTimeUS != msg.TimeUS {
			t.Errorf("TimeUS mismatch: msg.TimeUS=%d, fields[TimeUS]=%d", msg.TimeUS, fieldTimeUS)
		}
	}
}

func TestGetSlice(t *testing.T) {
	parser, err := NewParser(testFile)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	// Test slice by LineNo
	messages, err := parser.GetSlice(10, 20, SliceByLineNo)
	if err != nil {
		t.Fatalf("error getting slice by LineNo: %v", err)
	}
	if len(messages) != 10 {
		t.Errorf("expected 10 messages, got %d", len(messages))
	}
	for _, msg := range messages {
		if msg.LineNo < 10 || msg.LineNo >= 20 {
			t.Errorf("message LineNo %d outside range [10, 20)", msg.LineNo)
		}
	}

	// Test slice by TimeUS - get first IMU message to find valid time range
	if err := parser.SetFilter([]string{"IMU"}); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}
	firstMsg, err := parser.ReadMessage()
	if err != nil {
		t.Fatalf("error reading first IMU message: %v", err)
	}
	// Use time range around first message
	start := firstMsg.TimeUS
	end := start + 100000 // 0.1 second window
	messages, err = parser.GetSlice(start, end, SliceByTimeUS)
	if err != nil {
		t.Fatalf("error getting slice by TimeUS: %v", err)
	}
	if len(messages) == 0 {
		t.Error("expected some messages in time range")
	}
	for _, msg := range messages {
		if msg.TimeUS < start || msg.TimeUS >= end {
			t.Errorf("message TimeUS %d outside range [%d, %d)", msg.TimeUS, start, end)
		}
	}
}
