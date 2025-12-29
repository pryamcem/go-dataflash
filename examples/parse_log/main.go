package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pryamcem/go-dataflash"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: parse_log <logfile.bin>")
		os.Exit(1)
	}

	log.Println("Creating parser")
	parser, err := dataflash.NewParser(os.Args[1])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer parser.Close()

	// Filter to only get GPS messages
	log.Println("Set filters")
	if err := parser.SetFilter([]string{"GPS", "IMU", "TECS"}); err != nil {
		log.Fatalf("Error setting filter: %v", err)
	}

	// Read messages
	count := 0
	for {
		msg, err := parser.ReadMessage()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error reading message: %v", err)
		}

		fmt.Printf("%s #%d: ", msg.Name, count+1)
		// Print first few fields
		for k, v := range msg.Fields {
			fmt.Printf("%s=%v ", k, v)
		}
		fmt.Println()
		count++
		
		if count >= 10 {
			break
		}
	}
}
