package dataflash

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// formatSizes maps format characters to their byte sizes
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

// decodeValue decodes a single field value from b (which must start at the
// field's offset) given its format character. Returns nil for unknown formats.
func decodeValue(formatChar byte, b []byte) any {
	switch rune(formatChar) {
	case 'B':
		return b[0]
	case 'H':
		return binary.LittleEndian.Uint16(b)
	case 'I':
		return binary.LittleEndian.Uint32(b)
	case 'Q':
		return binary.LittleEndian.Uint64(b)
	case 'b':
		return int8(b[0])
	case 'h':
		return int16(binary.LittleEndian.Uint16(b))
	case 'i':
		return int32(binary.LittleEndian.Uint32(b))
	case 'q':
		return int64(binary.LittleEndian.Uint64(b))
	case 'f':
		return math.Float32frombits(binary.LittleEndian.Uint32(b))
	case 'd':
		return math.Float64frombits(binary.LittleEndian.Uint64(b))
	case 'c':
		return float64(int16(binary.LittleEndian.Uint16(b))) * 0.01
	case 'C':
		return float64(binary.LittleEndian.Uint16(b)) * 0.01
	case 'e':
		return float64(int32(binary.LittleEndian.Uint32(b))) * 0.01
	case 'E':
		return float64(binary.LittleEndian.Uint32(b)) * 0.01
	case 'L':
		return float64(int32(binary.LittleEndian.Uint32(b))) * 1e-7
	case 'n':
		return strings.TrimRight(string(b[:4]), "\x00")
	case 'N':
		return strings.TrimRight(string(b[:16]), "\x00")
	case 'Z':
		return strings.TrimRight(string(b[:64]), "\x00")
	default:
		return nil
	}
}

// ensureLayout lazily computes and caches the per-field column names, byte
// offsets, and format chars for the schema, so single-field lazy decode does
// not re-split the columns string or re-walk the format on every access.
func (s *Schema) ensureLayout() {
	if s.colNames != nil {
		return
	}
	names := parseColumns(s.Columns)
	offsets := make([]int, len(names))
	formats := make([]byte, len(names))
	off := 0
	for i := range names {
		if i >= len(s.Format) || s.Format[i] == 0 {
			offsets[i] = off
			continue
		}
		fc := s.Format[i]
		formats[i] = fc
		offsets[i] = off
		off += formatSizes[rune(fc)]
	}
	s.colNames = names
	s.colOffsets = offsets
	s.colFormats = formats
}

// DecodeMessageBody decodes a message body according to the provided schema.
// Returns a map of field names to their decoded values.
//
// If body is too short for the schema, decoding stops at the first field that
// does not fit: the fields decoded so far are returned together with a non-nil
// error, so callers can choose to use the partial result.
func DecodeMessageBody(body []byte, schema *Schema) (map[string]any, error) {
	data := make(map[string]any)
	columns := strings.Split(schema.Columns, ",")

	offset := 0

	for i := 0; i < len(schema.Format); i++ {
		dataType := schema.Format[i]

		// Stop at null terminator
		if dataType == 0 {
			break
		}

		// Stop if no more columns
		if i >= len(columns) {
			break
		}

		size := formatSizes[rune(dataType)]
		if offset+size > len(body) {
			return data, fmt.Errorf("message body truncated: field %q (format %q) needs %d bytes at offset %d, body has %d",
				columns[i], dataType, size, offset, len(body))
		}

		// Unknown format types decode to nil and are skipped
		if value := decodeValue(dataType, body[offset:]); value != nil {
			data[columns[i]] = value
		}

		offset += size
	}

	return data, nil
}
