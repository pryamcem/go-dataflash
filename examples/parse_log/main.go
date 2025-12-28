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

	// Filter to only get GPS messages
	parser.SetFilter([]string{"GPS"})

	// Read first 5 GPS messages
	count := 0
	for count < 5 {
		msg, err := parser.ReadMessage()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("GPS #%d: Lat=%v, Lng=%v, Alt=%v\n",
			count+1, msg.Fields["Lat"], msg.Fields["Lng"], msg.Fields["Alt"])
		count++
	}
}
