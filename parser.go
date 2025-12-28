package dataflash

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// DataFlash binary format constants
const (
	HEAD1      = 0xA3 // First magic byte
	HEAD2      = 0x95 // Second magic byte
	FMTType    = 128  // FMT message type
	FMTLength  = 89   // FMT message total length
	HeaderSize = 3    // Header size in bytes
)

// Parser reads and parses ArduPilot DataFlash binary logs.
type Parser struct {
	file    *os.File
	schemas map[uint8]*Schema
}

// NewParser creates a new parser for the given DataFlash log file.
// It performs a first pass to build the schema map from FMT messages.
func NewParser(filename string) (*Parser, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	p := &Parser{
		file:    file,
		schemas: make(map[uint8]*Schema),
	}

	// Pass 1: Build schema map from FMT messages
	if err := p.buildSchemas(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to build schemas: %w", err)
	}

	// Rewind for reading messages
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to rewind file: %w", err)
	}

	return p, nil
}

// Close closes the underlying file.
func (p *Parser) Close() error {
	return p.file.Close()
}

// GetSchemas returns a map of all message schemas found in the log.
func (p *Parser) GetSchemas() map[uint8]*Schema {
	return p.schemas
}

// ReadMessage reads and parses the next message from the log.
// Returns io.EOF when there are no more messages.
func (p *Parser) ReadMessage() (*Message, error) {
	msgType, err := p.readMessageHeader()
	if err != nil {
		return nil, err
	}

	schema, ok := p.schemas[msgType]
	if !ok {
		return nil, fmt.Errorf("unknown message type: %d", msgType)
	}

	// Read message body
	bodySize := int(schema.Length) - HeaderSize
	body := make([]byte, bodySize)
	if _, err := io.ReadFull(p.file, body); err != nil {
		return nil, err
	}

	// Decode message body
	fields, err := DecodeMessageBody(body, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	return &Message{
		Type:   msgType,
		Name:   schema.Name,
		Fields: fields,
	}, nil
}

// buildSchemas performs the first pass to read all FMT messages.
func (p *Parser) buildSchemas() error {
	for {
		msgType, err := p.readMessageHeader()
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			// Skip invalid headers
			continue
		}

		if msgType == FMTType {
			schema, err := p.decodeFMTMessage()
			if err != nil {
				return err
			}
			p.schemas[schema.Type] = schema
		} else {
			// Unknown message type - sync to next header
			if err := p.syncToNextHeader(); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					break
				}
				return err
			}
		}
	}

	return nil
}

// syncToNextHeader scans forward byte-by-byte to find the next valid message header.
// This is used when we encounter unknown message types during schema building.
func (p *Parser) syncToNextHeader() error {
	for {
		byte1 := make([]byte, 1)
		_, err := p.file.Read(byte1)
		if err != nil {
			return err
		}

		if byte1[0] == HEAD1 {
			// Found potential first magic byte, check second
			byte2 := make([]byte, 1)
			_, err := p.file.Read(byte2)
			if err != nil {
				return err
			}

			if byte2[0] == HEAD2 {
				// Found valid header! Seek back 2 bytes so next read gets the full header
				_, err := p.file.Seek(-2, io.SeekCurrent)
				return err
			}
			// Second byte didn't match, seek back 1 and continue
			p.file.Seek(-1, io.SeekCurrent)
		}
	}
}

// readMessageHeader reads and validates a 3-byte message header.
func (p *Parser) readMessageHeader() (uint8, error) {
	header := make([]byte, HeaderSize)
	_, err := io.ReadFull(p.file, header)
	if err != nil {
		return 0, err
	}

	if header[0] != HEAD1 || header[1] != HEAD2 {
		return 0, fmt.Errorf("invalid header")
	}

	return header[2], nil
}

// decodeFMTMessage reads and decodes a FMT message from the current file position.
func (p *Parser) decodeFMTMessage() (*Schema, error) {
	var schema Schema

	if err := binary.Read(p.file, binary.LittleEndian, &schema.Type); err != nil {
		return nil, fmt.Errorf("reading FMT type: %w", err)
	}
	if err := binary.Read(p.file, binary.LittleEndian, &schema.Length); err != nil {
		return nil, fmt.Errorf("reading FMT length: %w", err)
	}

	var err error
	schema.Name, err = readString(p.file, 4)
	if err != nil {
		return nil, err
	}
	schema.Format, err = readString(p.file, 16)
	if err != nil {
		return nil, err
	}
	schema.Columns, err = readString(p.file, 64)
	if err != nil {
		return nil, err
	}

	return &schema, nil
}

// readString reads a null-terminated string of maximum length from the file.
func readString(file *os.File, maxLen int) (string, error) {
	buf := make([]byte, maxLen)
	_, err := file.Read(buf)
	if err != nil {
		return "", err
	}

	// Find null terminator
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i]), nil
		}
	}
	return string(buf), nil
}
