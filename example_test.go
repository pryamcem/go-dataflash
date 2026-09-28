package dataflash_test

import (
	"fmt"
	"log"
	"os"

	dataflash "github.com/pryamcem/go-dataflash/v3"
)

func ExampleParser_Messages() {
	f, err := os.Open("testdata/testlog.bin")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	parser, err := dataflash.NewParser(f)
	if err != nil {
		log.Fatal(err)
	}
	if err := parser.SetFilter("GPS"); err != nil {
		log.Fatal(err)
	}

	count := 0
	for msg, err := range parser.Messages() {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(msg.Name, msg.TimeUS)
		count++
		if count >= 3 {
			break
		}
	}

	// Output:
	// GPS 44301587
	// GPS 44501358
	// GPS 44701065
}

func ExampleParser_SetFilter() {
	f, err := os.Open("testdata/testlog.bin")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	parser, err := dataflash.NewParser(f)
	if err != nil {
		log.Fatal(err)
	}
	if err := parser.SetFilter("GPS", "IMU"); err != nil {
		log.Fatal(err)
	}

	count := 0
	for msg, err := range parser.Messages() {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(msg.Name)
		count++
		if count >= 5 {
			break
		}
	}

	// Output:
	// IMU
	// IMU
	// IMU
	// IMU
	// IMU
}

func ExampleMessage_Get() {
	f, err := os.Open("testdata/testlog.bin")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	parser, err := dataflash.NewParser(f)
	if err != nil {
		log.Fatal(err)
	}
	if err := parser.SetFilter("GPS"); err != nil {
		log.Fatal(err)
	}

	msg, err := parser.ReadMessage()
	if err != nil {
		log.Fatal(err)
	}

	alt, ok := msg.Get("Alt")
	fmt.Println(alt, ok)

	// Output:
	// 62.14 true
}

func ExampleMessage_GetScaled() {
	f, err := os.Open("testdata/testlog.bin")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	parser, err := dataflash.NewParser(f)
	if err != nil {
		log.Fatal(err)
	}
	if err := parser.SetFilter("GPS"); err != nil {
		log.Fatal(err)
	}

	msg, err := parser.ReadMessage()
	if err != nil {
		log.Fatal(err)
	}

	sv, err := msg.GetScaled("Alt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v %s\n", sv.Value, sv.Unit)

	// Output:
	// 62.14 m
}
