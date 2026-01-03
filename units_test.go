package dataflash

import (
	"testing"
)

func TestSchemaHasUnitsAndMults(t *testing.T) {
	parser, err := NewParser(testFile)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	schemas := parser.GetSchemas()

	// Find a schema that should have units (most data messages do)
	foundWithUnits := false
	for _, schema := range schemas {
		if schema.Units != "" || schema.Mults != "" {
			foundWithUnits = true
			t.Logf("Schema %s has units=%q, mults=%q", schema.Name, schema.Units, schema.Mults)
			
			// Units and Mults should match Format length
			if len(schema.Units) > 0 && len(schema.Units) != len(schema.Format) {
				t.Errorf("%s: units length (%d) doesn't match format length (%d)", 
					schema.Name, len(schema.Units), len(schema.Format))
			}
			if len(schema.Mults) > 0 && len(schema.Mults) != len(schema.Format) {
				t.Errorf("%s: mults length (%d) doesn't match format length (%d)", 
					schema.Name, len(schema.Mults), len(schema.Format))
			}
			break
		}
	}

	if !foundWithUnits {
		t.Skip("No schemas with units/mults found in test log")
	}
}

func TestGetScaled(t *testing.T) {
	parser, err := NewParser(testFile)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	// Read until we find a message with TimeUS
	if err := parser.SetFilter("IMU"); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}

	msg, err := parser.ReadMessage()
	if err != nil {
		t.Fatalf("error reading message: %v", err)
	}

	// Test GetScaled on TimeUS
	value, unit, err := msg.GetScaled("TimeUS")
	if err != nil {
		t.Fatalf("GetScaled failed: %v", err)
	}

	// TimeUS should be scaled from microseconds to seconds
	if unit != "seconds" && unit != "s" {
		t.Errorf("expected TimeUS unit to be seconds, got %q", unit)
	}

	// Value should be float64 for scaled fields
	floatVal, ok := value.(float64)
	if !ok {
		t.Errorf("expected float64, got %T", value)
	}

	// Scaled value should be reasonable (a few seconds, not millions)
	if floatVal < 0 || floatVal > 1000 {
		t.Errorf("scaled TimeUS seems wrong: %f %s", floatVal, unit)
	}

	t.Logf("TimeUS: raw=%v, scaled=%v %s", msg.Fields["TimeUS"], value, unit)
}

func TestGetScaledFields(t *testing.T) {
	parser, err := NewParser(testFile)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	if err := parser.SetFilter("GPS"); err != nil {
		t.Skip("No GPS messages in log")
	}

	msg, err := parser.ReadMessage()
	if err != nil {
		t.Skip("Could not read GPS message")
	}

	// Get all scaled fields
	scaled := msg.GetScaledFields()

	if len(scaled) == 0 {
		t.Error("expected some scaled fields")
	}

	// Count numeric fields in original message
	numericFields := 0
	for _, v := range msg.Fields {
		if _, err := toFloat64(v); err == nil {
			numericFields++
		}
	}

	// GetScaledFields should return all numeric fields
	if len(scaled) != numericFields {
		t.Errorf("expected %d numeric fields, got %d", numericFields, len(scaled))
	}

	// Log some example fields
	for name, sv := range scaled {
		if sv.Unit != "" {
			t.Logf("%s: %v %s", name, sv.Value, sv.Unit)
		}
	}

	// Verify fields without scaling multipliers still appear
	if status, ok := scaled["Status"]; ok {
		// Status typically has no scaling (multiplier '-' or '0')
		t.Logf("Status (no scaling): %v (type: %T)", status.Value, status.Value)
	}
}

func TestGetScaledInvalidField(t *testing.T) {
	parser, err := NewParser(testFile)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	if err := parser.SetFilter("IMU"); err != nil {
		t.Fatalf("failed to set filter: %v", err)
	}

	msg, err := parser.ReadMessage()
	if err != nil {
		t.Fatalf("error reading message: %v", err)
	}

	// Try to get a field that doesn't exist
	_, _, err = msg.GetScaled("NonExistentField")
	if err == nil {
		t.Error("expected error for non-existent field")
	}
}
