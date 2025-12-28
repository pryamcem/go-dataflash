package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
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

var formatSizes = map[rune]int{
	'B': 1,  // uint8
	'b': 1,  // int8
	'H': 2,  // uint16
	'h': 2,  // int16
	'I': 4,  // uint32
	'i': 4,  // int32
	'f': 4,  // float32
	'Q': 8,  // uint64
	'q': 8,  // int64
	'd': 8,  // float64
	'c': 2,  // int16 * 100 (scaled)
	'C': 2,  // uint16 * 100 (scaled)
	'e': 4,  // int32 * 100 (scaled)
	'E': 4,  // uint32 * 100 (scaled)
	'L': 4,  // int32 * 1e-7 (lat/lon)
	'n': 4,  // char[4]
	'N': 16, // char[16]
	'Z': 64, // char[64]
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
			fmtMsg, _ := decodeFMTMessage(file)
			schemas[fmtMsg.Type] = fmtMsg
			fmt.Print(".")
		} else {
			file.Seek(86, io.SeekCurrent)
		}
	}

	// Pass 2
	file.Seek(0, io.SeekStart)

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

		body := make([]byte, msgFormat.Length-3)
		_, err = io.ReadFull(file, body)
		if err != nil {
			break
		}
		data, _ := decodeMessageBody(body, msgFormat)

		// Temp test
		if msgFormat.Name == "GPS" {
			fmt.Println(data)
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

func decodeFMTMessage(file *os.File) (*FMTMessage, error) {
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

func decodeMessageBody(body []byte, schema *FMTMessage) (map[string]any, error) {
	data := make(map[string]any)
	columns := strings.Split(schema.Columns, ",")

	offset := 0 // Track position in body bytes

	for i, dataType := range schema.Format {
		// Stop at null terminator
		if dataType == 0 {
			break
		}

		// Stop if no more columns
		if i >= len(columns) {
			break
		}

		columnName := columns[i]

		// Decode based on format type
		var value any
		switch dataType {
		// Unsigned integers
		case 'B': // uint8
			value = body[offset]
		case 'H': // uing16
			value = binary.LittleEndian.Uint16(body[offset:])
		case 'I': // uint32
			value = binary.LittleEndian.Uint32(body[offset:])
		case 'Q': // uint64
			value = binary.LittleEndian.Uint64(body[offset:])

		// Signed integers
		case 'b': // int8
			value = int8(body[offset])
		case 'h': // int16
			value = int16(binary.LittleEndian.Uint16(body[offset:]))
		case 'i': // int32
			value = int32(binary.LittleEndian.Uint32(body[offset:]))
		case 'q': // int64
			value = int64(binary.LittleEndian.Uint64(body[offset:]))

		// Floats
		case 'f': // float32
			bits := binary.LittleEndian.Uint32(body[offset:])
			value = math.Float32frombits(bits)
		case 'd': // float64
			bits := binary.LittleEndian.Uint64(body[offset:])
			value = math.Float64frombits(bits)

		// Scaled values
		case 'c': // int16 * 100
			raw := int16(binary.LittleEndian.Uint16(body[offset:]))
			value = float64(raw) * 0.01
		case 'C': // uint16 * 100
			raw := binary.LittleEndian.Uint16(body[offset:])
			value = float64(raw) * 0.01
		case 'e': // int32 * 100
			raw := int32(binary.LittleEndian.Uint32(body[offset:]))
			value = float64(raw) * 0.01
		case 'E': // uint32 * 100
			raw := binary.LittleEndian.Uint32(body[offset:])
			value = float64(raw) * 0.01
		case 'L': // int32 * 1e-7 (latitude/longitude)
			raw := int32(binary.LittleEndian.Uint32(body[offset:]))
			value = float64(raw) * 1e-7

		// Strings
		case 'n': // char[4]
			value = strings.TrimRight(string(body[offset:offset+4]), "\x00")
		case 'N': // char[16]
			value = strings.TrimRight(string(body[offset:offset+16]), "\x00")
		case 'Z': // char[64]
			value = strings.TrimRight(string(body[offset:offset+64]), "\x00")
		default: // Unknown
			offset += formatSizes[dataType]
			continue
		}

		// Store the value
		data[columnName] = value

		// Move offset forward
		offset += formatSizes[dataType]
	}

	return data, nil
}
