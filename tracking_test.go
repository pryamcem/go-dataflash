package dataflash

import (
	"testing"
)

func TestMessageTracking(t *testing.T) {
	parser, err := NewParser("log_29_2025-12-13-16-08-22.bin")
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
	parser, err := NewParser("log_29_2025-12-13-16-08-22.bin")
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

	// Test slice by TimeUS
	if err := parser.SetFilter([]string{"IMU"}); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}
	messages, err = parser.GetSlice(2800000, 2900000, SliceByTimeUS)
	if err != nil {
		t.Fatalf("error getting slice by TimeUS: %v", err)
	}
	if len(messages) == 0 {
		t.Error("expected some messages in time range")
	}
	for _, msg := range messages {
		if msg.TimeUS < 2800000 || msg.TimeUS >= 2900000 {
			t.Errorf("message TimeUS %d outside range", msg.TimeUS)
		}
	}
}
