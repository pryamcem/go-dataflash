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
		log.Fatal(err)
	}
	defer parser.Close()

	// Filter for GPS messages which have many fields with units
	log.Println("Set filters")
	if err := parser.SetFilter("GPS"); err != nil {
		log.Fatal(err)
	}

	log.Println("10 Scaled Fields")
	for count := range 10 {
		msg, err := parser.ReadMessage()
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		log.Println(msg.Fields)

		scaled := msg.GetScaledFields()
		fmt.Println(scaled)

		if count > 10 {
			break
		}
	}
}
