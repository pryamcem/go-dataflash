package dataflash

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	readBufferSize = 64 * 1024
	maxBodySize    = 256
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
	source      io.ReadSeeker
	reader      *bufio.Reader
	schemas     map[uint8]*Schema
	filterTypes map[uint8]bool
	lineNo      int64 // Current message sequence number

	// Pre-allocated buffers
	headerBuf [HeaderSize]byte
	bodyBuf   [maxBodySize]byte
	syncBuf   [1]byte
}

// NewParser creates a new parser for the given source.
// The source must be seekable as the parser performs two passes.
func NewParser(source io.ReadSeeker) (*Parser, error) {
	p := &Parser{
		source:  source,
		reader:  bufio.NewReaderSize(source, readBufferSize),
		schemas: make(map[uint8]*Schema),
	}

	// Pass 1: Build schema map from FMT messages
	if err := p.buildSchemas(); err != nil {
		return nil, fmt.Errorf("failed to build schemas: %w", err)
	}

	// Rewind for reading messages
	if err := p.rewind(); err != nil {
		return nil, fmt.Errorf("failed to rewind source: %w", err)
	}

	return p, nil
}

// Close is a no-op. The caller is responsible for closing the source.
func (p *Parser) Close() error {
	return nil
}

// GetSchemas returns a copy of all message schemas found in the log.
func (p *Parser) GetSchemas() map[uint8]*Schema {
	result := make(map[uint8]*Schema, len(p.schemas))
	for k, v := range p.schemas {
		schema := *v
		result[k] = &schema
	}
	return result
}

// ReadMessage reads and parses the next message from the log.
// Returns io.EOF when there are no more messages.
func (p *Parser) ReadMessage() (*Message, error) {
	for {
		msgType, err := p.readMessageHeader()
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, err
		}
		if err != nil {
			// Invalid header - try to sync to next valid header
			if syncErr := p.syncToNextHeader(); syncErr != nil {
				if syncErr == io.EOF || syncErr == io.ErrUnexpectedEOF {
					return nil, syncErr
				}
				// Continue trying to read next message
			}
			continue
		}

		// Check if we have schema for this message type
		schema, ok := p.schemas[msgType]
		if !ok {
			// Unknown message type - sync to next header
			if syncErr := p.syncToNextHeader(); syncErr != nil {
				if syncErr == io.EOF || syncErr == io.ErrUnexpectedEOF {
					return nil, syncErr
				}
			}
			continue
		}

		// Increment line number for every message
		p.lineNo++

		// Check filter before reading body
		if p.filterTypes != nil && !p.filterTypes[msgType] {
			bodySize := int(schema.Length) - HeaderSize
			p.reader.Discard(bodySize)
			continue
		}

		// Read message body using pre-allocated buffer
		bodySize := int(schema.Length) - HeaderSize
		body := p.bodyBuf[:bodySize]
		if _, err := io.ReadFull(p.reader, body); err != nil {
			return nil, err
		}

		// Decode message body
		fields, err := DecodeMessageBody(body, schema)
		if err != nil {
			return nil, fmt.Errorf("failed to decode message: %w", err)
		}

		// Extract TimeUS if available
		timeUS := int64(0)
		if val, ok := fields["TimeUS"]; ok {
			switch v := val.(type) {
			case int64:
				timeUS = v
			case uint64:
				timeUS = int64(v)
			}
		}

		return &Message{
			Type:   msgType,
			Name:   schema.Name,
			Fields: fields,
			LineNo: p.lineNo,
			TimeUS: timeUS,
			schema: schema,
		}, nil
	}
}

// SetFilter restricts parsing to the given message names.
// Automatically rewinds so all messages are available from the start.
// Passing no names clears the filter and all message types are returned.
// Returns an error if any name does not match a message type in the log.
func (p *Parser) SetFilter(names ...string) error {
	if len(names) == 0 {
		p.filterTypes = nil
		return p.rewind()
	}

	p.filterTypes = make(map[uint8]bool)
	var invalidNames []string

	for _, name := range names {
		found := false
		for typ, schema := range p.schemas {
			if schema.Name == name {
				p.filterTypes[typ] = true
				found = true
				break
			}
		}
		if !found {
			invalidNames = append(invalidNames, name)
		}
	}

	if len(p.filterTypes) == 0 {
		return fmt.Errorf("no valid message types found in filter: %v", names)
	}

	if len(invalidNames) > 0 {
		return fmt.Errorf("invalid message types in filter: %v", invalidNames)
	}

	// Rewind to start so filter applies from beginning
	return p.rewind()
}

// Rewind resets the source position to the beginning.
// Useful for re-reading messages or starting a new iteration.
func (p *Parser) Rewind() error {
	return p.rewind()
}

// rewind is the internal helper that resets file position and buffered reader.
func (p *Parser) rewind() error {
	p.lineNo = 0
	if _, err := p.source.Seek(0, io.SeekStart); err != nil {
		return err
	}
	p.reader.Reset(p.source)
	return nil
}

// SliceType specifies how to slice the log.
type SliceType int

const (
	SliceByLineNo SliceType = iota
	SliceByTimeUS
)

// GetSlice returns messages within the specified range.
// start and end values are interpreted based on sliceType (LineNo or TimeUS).
// The returned messages are those where start <= value < end.
func (p *Parser) GetSlice(start, end int64, sliceType SliceType) ([]*Message, error) {
	if err := p.Rewind(); err != nil {
		return nil, err
	}

	var messages []*Message
	for {
		msg, err := p.ReadMessage()
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}

		var value int64
		switch sliceType {
		case SliceByLineNo:
			value = msg.LineNo
		case SliceByTimeUS:
			value = msg.TimeUS
		default:
			return nil, fmt.Errorf("invalid slice type: %d", sliceType)
		}

		if value >= start && value < end {
			messages = append(messages, msg)
		}

		// Early exit if we've passed the end
		if value >= end {
			break
		}
	}

	return messages, nil
}

// buildSchemas performs the first pass to read all FMT and FMTU messages.
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
		} else if schema, exists := p.schemas[msgType]; exists && schema.Name == "FMTU" {
			// Decode FMTU message to get units and multipliers
			bodySize := int(schema.Length) - HeaderSize
			body := p.bodyBuf[:bodySize]
			if _, err := io.ReadFull(p.reader, body); err != nil {
				continue
			}

			fields, err := DecodeMessageBody(body, schema)
			if err != nil {
				// Skip malformed FMTU messages
				continue
			}

			// Extract FmtType, UnitIds, MultIds fields
			fmtType, ok := fields["FmtType"].(uint8)
			if !ok {
				continue
			}
			unitIds, ok := fields["UnitIds"].(string)
			if !ok {
				continue
			}
			multIds, ok := fields["MultIds"].(string)
			if !ok {
				continue
			}

			// Update the corresponding schema with units and multipliers
			if targetSchema, exists := p.schemas[fmtType]; exists {
				targetSchema.Units = unitIds
				targetSchema.Mults = multIds
				p.schemas[fmtType] = targetSchema
			}
		} else if schema, exists := p.schemas[msgType]; exists {
			// Known non-FMT/FMTU message: skip its body using the schema's
			// recorded length. Falling back to syncToNextHeader here would
			// byte-scan for HEAD1/HEAD2 inside the payload, which can match
			// by coincidence and desync the parser - corrupting later FMT
			// records (and silently dropping the schemas they define).
			bodySize := int(schema.Length) - HeaderSize
			if bodySize < 0 {
				bodySize = 0
			}
			if _, err := p.reader.Discard(bodySize); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					break
				}
				return err
			}
		} else {
			// Truly unknown message type (no FMT seen yet) - sync to next header.
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

// syncToNextHeader scans forward to find the next valid message header.
// This is used when we encounter unknown message types during schema building.
// Uses Peek and Discard to avoid UnreadByte issues with buffered I/O.
func (p *Parser) syncToNextHeader() error {
	for {
		// Peek at the next 2 bytes to check for header pattern
		peeked, err := p.reader.Peek(2)
		if err != nil {
			return err
		}

		if peeked[0] == HEAD1 && peeked[1] == HEAD2 {
			// Found valid header! Don't consume it - let readMessageHeader do that
			return nil
		}

		// Not a header, skip one byte and try again
		p.reader.Discard(1)
	}
}

// readMessageHeader reads and validates a 3-byte message header.
func (p *Parser) readMessageHeader() (uint8, error) {
	_, err := io.ReadFull(p.reader, p.headerBuf[:])
	if err != nil {
		return 0, err
	}

	if p.headerBuf[0] != HEAD1 || p.headerBuf[1] != HEAD2 {
		return 0, fmt.Errorf("invalid header")
	}

	return p.headerBuf[2], nil
}

// decodeFMTMessage reads and decodes a FMT message from the current file position.
func (p *Parser) decodeFMTMessage() (*Schema, error) {
	var schema Schema

	if err := binary.Read(p.reader, binary.LittleEndian, &schema.Type); err != nil {
		return nil, fmt.Errorf("reading FMT type: %w", err)
	}
	if err := binary.Read(p.reader, binary.LittleEndian, &schema.Length); err != nil {
		return nil, fmt.Errorf("reading FMT length: %w", err)
	}

	var err error
	schema.Name, err = readString(p.reader, 4)
	if err != nil {
		return nil, err
	}
	schema.Format, err = readString(p.reader, 16)
	if err != nil {
		return nil, err
	}
	schema.Columns, err = readString(p.reader, 64)
	if err != nil {
		return nil, err
	}

	return &schema, nil
}

// readString reads a null-terminated string of maximum length from the reader.
func readString(r io.Reader, maxLen int) (string, error) {
	buf := make([]byte, maxLen)
	_, err := io.ReadFull(r, buf)
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
