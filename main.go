package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	HEAD1 = 0xA3
	HEAD2 = 0x95
)

type FMTMessage struct {
	Type    uint8
	Length  uint8
	Name    string
	Format  string
	Columns string
}

func main() {
	file, err := os.Open("log_29_2025-12-13-16-08-22.bin")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	schemas := make(map[uint8]*FMTMessage)
	// Pass 1
	for {
		msgType, err := readMessageHeader(file)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			continue // Invalid header, skip
		}

		if msgType == 128 {
			fmtMsg, _ := parseFMTMessage(file)
			schemas[fmtMsg.Type] = fmtMsg
		} else {
			file.Seek(86, io.SeekCurrent)
		}
	}

	// Pass 2
	file.Seek(0, io.SeekStart)
	msgCounts := make(map[string]int)

	for {
		msgType, err := readMessageHeader(file)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			continue
		}

		msgFormat, ok := schemas[msgType]
		if !ok {
			continue
		}

		msgCounts[msgFormat.Name]++
		file.Seek(int64(msgFormat.Length-3), io.SeekCurrent)
	}
	for _, f := range schemas {
		if f.Name == "GPS" {
			fmt.Printf("GPS Type: %d\n", f.Type)
			fmt.Printf("GPS Format: %s\n", f.Format)
			fmt.Printf("GPS Columns: %s\n", f.Columns)
			fmt.Printf("GPS Length: %d\n", f.Length)
		}
	}
}

func readMessageHeader(file *os.File) (uint8, error) {
	header := make([]byte, 3)
	_, err := io.ReadFull(file, header)
	if err != nil {
		return 0, err
	}

	if header[0] != HEAD1 || header[1] != HEAD2 {
		return 0, fmt.Errorf("invalid header")
	}

	return header[2], nil // Return message type
}

func readString(file *os.File, maxLen int) (string, error) {
	buf := make([]byte, maxLen)
	_, err := file.Read(buf)
	if err != nil {
		return "", err
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i]), nil
		}
	}
	return string(buf), nil
}

func parseFMTMessage(file *os.File) (*FMTMessage, error) {
	var msg FMTMessage
	if err := binary.Read(file, binary.LittleEndian, &msg.Type); err != nil {
		return nil, fmt.Errorf("reading FMT type: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &msg.Length); err != nil {
		return nil, fmt.Errorf("reading FMT length: %w", err)
	}

	var err error
	msg.Name, err = readString(file, 4)
	if err != nil {
		return nil, err
	}
	msg.Format, err = readString(file, 16)
	if err != nil {
		return nil, err
	}
	msg.Columns, err = readString(file, 64)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}
