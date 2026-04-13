package dataflash

import (
	"bytes"
	"io"
	"os"
	"testing"
)

const (
	// Origin: https://discuss.ardupilot.org/t/vtol-crash-after-transition-to-fbwa/138484
	testFile = "testdata/testlog.bin"
)

func TestNewParser(t *testing.T) {
	// Read file into memory
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	// Create bytes.Reader which implements io.ReadSeeker
	source := bytes.NewReader(data)

	// Create parser from source
	parser, err := NewParser(source)
	if err != nil {
		t.Fatalf("failed to create parser from source: %v", err)
	}
	defer parser.Close()

	// Verify we can read a message
	msg, err := parser.ReadMessage()
	if err != nil {
		t.Fatalf("error reading message: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
}

func TestRewindNonFileSource(t *testing.T) {
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	parser, err := NewParser(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	// SetFilter should rewind and return only GPS messages
	if err := parser.SetFilter("GPS"); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}
	msg, err := parser.ReadMessage()
	if err != nil {
		t.Fatalf("error reading message after SetFilter: %v", err)
	}
	if msg.Name != "GPS" {
		t.Errorf("expected GPS, got %s", msg.Name)
	}

	// GetSlice should rewind internally and return messages in range
	parser.ClearFilter()
	messages, err := parser.GetSlice(1, 5, SliceByLineNo)
	if err != nil {
		t.Fatalf("error getting slice: %v", err)
	}
	if len(messages) == 0 {
		t.Error("expected messages in slice, got none")
	}
	for _, m := range messages {
		if m.LineNo < 1 || m.LineNo >= 5 {
			t.Errorf("message LineNo %d outside range [1, 5)", m.LineNo)
		}
	}
}

func TestCloseNonCloserSource(t *testing.T) {
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	// bytes.Reader does not implement io.Closer
	parser, err := NewParser(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	if err := parser.Close(); err != nil {
		t.Errorf("expected nil error closing non-closer source, got: %v", err)
	}
}

func TestParserFilter(t *testing.T) {
	f, err := os.Open("testdata/testlog.bin")
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()
	parser, err := NewParser(f)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Set filter to only GPS
	if err := parser.SetFilter("GPS"); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}

	// Read 10 messages
	for range 10 {
		msg, err := parser.ReadMessage()
		if err == io.EOF || err == io.ErrUnexpectedEOF {
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
	f, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()
	parser, err := NewParser(f)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Try to set filter with all invalid names
	err = parser.SetFilter("INVALID", "NOTEXIST")
	if err == nil {
		t.Fatal("expected error for invalid filter names, got nil")
	}

	// Try to set filter with mix of valid and invalid
	err = parser.SetFilter("GPS", "INVALID")
	if err == nil {
		t.Fatal("expected error for invalid filter name, got nil")
	}
	if err.Error() != "invalid message types in filter: [INVALID]" {
		t.Errorf("unexpected error message: %v", err)
	}

	// Set valid filter
	err = parser.SetFilter("GPS")
	if err != nil {
		t.Fatalf("unexpected error for valid filter: %v", err)
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
	f, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()
	parser, err := NewParser(f)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Read 5 GPS messages
	if err := parser.SetFilter("GPS"); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := parser.ReadMessage(); err != nil {
			t.Fatalf("error reading GPS: %v", err)
		}
	}

	// Change filter to IMU - should rewind automatically
	if err := parser.SetFilter("IMU"); err != nil {
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
	f, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()
	parser, err := NewParser(f)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Filter for IMU which should have TimeUS
	if err := parser.SetFilter("IMU"); err != nil {
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
	f, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer f.Close()
	parser, err := NewParser(f)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

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
	if err := parser.SetFilter("IMU"); err != nil {
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
