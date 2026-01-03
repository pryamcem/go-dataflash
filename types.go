package dataflash

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
}

// Message represents a parsed DataFlash message with its decoded field values.
type Message struct {
	Type   uint8          // Message type ID
	Name   string         // Message name
	Fields map[string]any // Decoded field values
	LineNo int64          // Message sequence number in the log
	TimeUS int64          // Microseconds since boot (0 if not available)
	schema *Schema        // Reference to schema for unit/mult lookups
}

// ScaledValue represents a field value with its unit
type ScaledValue struct {
	Value any    // Field value (preserves original type when no scaling needed)
	Unit  string // Unit name (e.g., "seconds", "meters")
}
