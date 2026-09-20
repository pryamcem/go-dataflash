package dataflash

import (
	"encoding/binary"
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
func DecodeMessageBody(body []byte, schema *Schema) (map[string]any, error) {
	data := make(map[string]any)
	columns := strings.Split(schema.Columns, ",")

	offset := 0

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
		case 'H': // uint16
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

		default: // Unknown format type
			offset += formatSizes[dataType]
			continue
		}

		// Store the decoded value
		data[columnName] = value

		// Move offset forward by the size of this field
		offset += formatSizes[dataType]
	}

	return data, nil
}
