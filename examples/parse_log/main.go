package main

import (
	"fmt"
	"io"
	"os"

	"github.com/pryamcem/go-dataflash"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: parse_log <logfile.bin>")
		os.Exit(1)
	}

	parser, err := dataflash.NewParser(os.Args[1])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer parser.Close()

	for {
		msg, err := parser.ReadMessage()
		if err == io.EOF {
			break // No more messages
		}
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if msg.Name == "GPS" {
			fmt.Printf("GPS: %v\n", msg.Fields)
		}
	}
}
