package dataflash

import (
	"fmt"
	"io"
	"testing"
	"time"
)

func TestParserTiming(t *testing.T) {
	start := time.Now()
	
	// Step 1: Create parser (Pass 1 - build schemas)
	t1 := time.Now()
	parser, err := NewParser("log_29_2025-12-13-16-08-22.bin")
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()
	t2 := time.Now()
	fmt.Printf("Pass 1 (build schemas): %v\n", t2.Sub(t1))

	// Step 2: Set filter
	t3 := time.Now()
	if err := parser.SetFilter([]string{"GPS"}); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}
	t4 := time.Now()
	fmt.Printf("Set filter: %v\n", t4.Sub(t3))

	// Step 3: Read 10 GPS messages
	t5 := time.Now()
	count := 0
	for count < 10 {
		_, err := parser.ReadMessage()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("error reading message: %v", err)
		}
		count++
	}
	t6 := time.Now()
	fmt.Printf("Read 10 GPS messages: %v\n", t6.Sub(t5))
	
	fmt.Printf("Total time: %v\n", time.Since(start))
}
