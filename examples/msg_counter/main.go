package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pryamcem/go-dataflash/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: parse_log <logfile.bin>")
		os.Exit(1)
	}

	log.Println("Creating parser")
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	parser, err := dataflash.NewParser(f)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Filter to only get GPS messages
	log.Println("Set filters")
	if err := parser.SetFilter("GPS", "IMU", "POS"); err != nil {
		log.Fatalf("Error setting filter: %v", err)
	}

	// Read messages
	messageCount := make(map[string]int32, 3)
	for {
		msg, err := parser.ReadMessage()
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			log.Fatalf("Error reading message: %v", err)
		}
		messageCount[msg.Name]++
	}
	for name, count := range messageCount {
		fmt.Println(name, count)
	}
}
