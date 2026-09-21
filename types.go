package dataflash

import "encoding/binary"

// Schema represents a message format definition (FMT message).
// It describes how to decode a specific message type.
type Schema struct {
	Type    uint8  // Message type ID
	Length  uint8  // Total message length including 3-byte header
	Name    string // Message name (e.g., "GPS", "IMU")
	Format  string // Format string (e.g., "QBBIHBcLLeffffB")
	Columns string // Comma-separated column names
	Units   string // Unit identifiers per field (from FMTU)
	Mults   string // Multiplier identifiers per field (from FMTU)

	// Cached layout for lazy single-field decode (populated by ensureLayout).
	colNames   []string
	colOffsets []int
	colFormats []byte
}

// Message represents a parsed DataFlash message. Field values are decoded
// lazily: the raw body is retained and fields are decoded on demand via Get,
// or all at once via Fields. A Message caches decoded fields, so a single
// Message must not be used from multiple goroutines at once.
type Message struct {
	Type   uint8   // Message type ID
	Name   string  // Message name
	LineNo int64   // Message sequence number in the log
	TimeUS int64   // Microseconds since boot (0 if not available)
	schema *Schema // Reference to schema for unit/mult lookups

	body   []byte         // raw message body, retained for lazy decode
	fields map[string]any // cached full decode (populated by Fields)
}

// Get decodes and returns a single field value by name, without building the
// full field map. The bool is false if the field is absent or undecodable.
//
// Get is meant for reading one or a few fields per message. Each call scans the
// column list and boxes the value, so to read most or all fields call Fields
// once instead of calling Get for every column.
func (m *Message) Get(field string) (any, bool) {
	if m.schema == nil || m.body == nil {
		return nil, false
	}
	m.schema.ensureLayout()
	for i, name := range m.schema.colNames {
		if name != field {
			continue
		}
		fc := m.schema.colFormats[i]
		off := m.schema.colOffsets[i]
		if fc == 0 || off+formatSizes[rune(fc)] > len(m.body) {
			return nil, false
		}
		v := decodeValue(fc, m.body[off:])
		return v, v != nil
	}
	return nil, false
}

// decodeTimeUS populates m.TimeUS directly from the body without boxing,
// matching the original behaviour (only 64-bit TimeUS formats are recognised).
func (m *Message) decodeTimeUS() {
	if m.schema == nil || m.body == nil {
		return
	}
	m.schema.ensureLayout()
	for i, name := range m.schema.colNames {
		if name != "TimeUS" {
			continue
		}
		off := m.schema.colOffsets[i]
		fc := m.schema.colFormats[i]
		if off+8 > len(m.body) {
			return
		}
		switch rune(fc) {
		case 'Q', 'q':
			m.TimeUS = int64(binary.LittleEndian.Uint64(m.body[off:]))
		}
		return
	}
}

// Fields decodes (once, then caches) and returns all field values as a map.
// Prefer Get when you only need a few fields. If the body is shorter than the
// schema requires, the fields that fit are returned and the rest are omitted.
func (m *Message) Fields() map[string]any {
	if m.fields != nil {
		return m.fields
	}
	if m.schema != nil && m.body != nil {
		m.fields, _ = DecodeMessageBody(m.body, m.schema)
	}
	if m.fields == nil {
		m.fields = map[string]any{}
	}
	return m.fields
}

// Stats describes how much of the log the parser had to skip while reading.
// All values are zero (or false) for an undamaged log. Messages skipped on
// purpose by a filter are not counted.
type Stats struct {
	SkippedBytes   int64 // Bytes discarded without producing a message (including the bad or unknown 3-byte header)
	Resyncs        int64 // Times the parser searched forward for the next valid header
	InvalidHeaders int64 // Headers that did not start with the magic bytes
	UnknownTypes   int64 // Headers whose message type has no FMT definition in the log
	Truncated      bool  // The log ended in the middle of a message (the read still ends with io.EOF)
}

// ScaledValue represents a field value with its unit
type ScaledValue struct {
	Value any    // Field value (preserves original type when no scaling needed)
	Unit  string // Unit name (e.g., "seconds", "meters")
}
