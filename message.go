package dataflash

import (
	"fmt"
	"math"
)

// GetScaled returns the value and unit for a field.
// For fields whose format character includes scaling (c, C, e, E, L), the original value is returned.
// For other formats, the FMTU multiplier is applied if present (returns float64).
// Returns an error if the field doesn't exist.
func (m *Message) GetScaled(field string) (any, string, error) {
	if m.schema == nil {
		return nil, "", fmt.Errorf("no schema available for message")
	}

	// Get value from fields
	value, exists := m.Fields[field]
	if !exists {
		return nil, "", fmt.Errorf("field %q not found in message", field)
	}

	// Find field index in schema
	fieldIndex := -1
	columns := parseColumns(m.schema.Columns)
	for i, col := range columns {
		if col == field {
			fieldIndex = i
			break
		}
	}

	if fieldIndex == -1 {
		return nil, "", fmt.Errorf("field %q not found in schema", field)
	}

	// Get format character to determine if scaling was already applied during decoding
	formatChar := '-'
	if fieldIndex < len(m.schema.Format) {
		formatChar = rune(m.schema.Format[fieldIndex])
	}

	// Get unit and multiplier characters
	unitChar := '-'
	multChar := '-'
	if fieldIndex < len(m.schema.Units) {
		unitChar = rune(m.schema.Units[fieldIndex])
	}
	if fieldIndex < len(m.schema.Mults) {
		multChar = rune(m.schema.Mults[fieldIndex])
	}

	var scaledValue any = value

	// Only apply FMTU multiplier if format character doesn't already include scaling
	// Formats with built-in scaling: c, C, e, E, L
	if !formatHasScaling(formatChar) && multChar != '-' && multChar != '?' && multChar != '0' {
		// Need to apply multiplier - convert to float64
		floatValue, err := toFloat64(value)
		if err != nil {
			// Not numeric, return original value with unit
			return value, getUnitName(unitChar), nil
		}
		multiplier := getMultiplier(multChar)
		scaledValue = floatValue * multiplier
	}

	// Get unit name
	unit := getUnitName(unitChar)

	return scaledValue, unit, nil
}

// GetScaledFields returns a map of all fields with their units.
// For fields whose format character includes scaling (c, C, e, E, L), original values are preserved.
// For other numeric formats, the FMTU multiplier is applied if present (converted to float64).
// Non-numeric fields are included as-is.
func (m *Message) GetScaledFields() map[string]ScaledValue {
	result := make(map[string]ScaledValue)

	if m.schema == nil {
		return result
	}

	columns := parseColumns(m.schema.Columns)
	for i, col := range columns {
		// Get value
		value, exists := m.Fields[col]
		if !exists {
			continue
		}

		// Get format character
		formatChar := '-'
		if i < len(m.schema.Format) {
			formatChar = rune(m.schema.Format[i])
		}

		// Get unit and multiplier characters
		unitChar := '-'
		multChar := '-'
		if i < len(m.schema.Units) {
			unitChar = rune(m.schema.Units[i])
		}
		if i < len(m.schema.Mults) {
			multChar = rune(m.schema.Mults[i])
		}

		var scaledValue any = value

		// Only apply FMTU multiplier if format character doesn't already include scaling
		if !formatHasScaling(formatChar) && multChar != '-' && multChar != '?' && multChar != '0' {
			// Try to apply multiplier
			if floatValue, err := toFloat64(value); err == nil {
				multiplier := getMultiplier(multChar)
				scaledValue = floatValue * multiplier
			}
			// If conversion fails, keep original value
		}

		// Get unit name
		unit := getUnitName(unitChar)

		result[col] = ScaledValue{
			Value: scaledValue,
			Unit:  unit,
		}
	}

	return result
}

// toFloat64 converts various numeric types to float64
func toFloat64(val any) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		// Check for overflow
		if v > math.MaxInt64 {
			return float64(v), nil
		}
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

// formatHasScaling returns true if the format character already includes scaling during decode
func formatHasScaling(formatChar rune) bool {
	switch formatChar {
	case 'c', 'C', 'e', 'E', 'L':
		return true
	default:
		return false
	}
}

// parseColumns splits a comma-separated column string into individual field names
func parseColumns(columns string) []string {
	if columns == "" {
		return nil
	}

	var result []string
	var current string
	
	for _, ch := range columns {
		if ch == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	
	// Add last field
	if current != "" {
		result = append(result, current)
	}
	
	return result
}
