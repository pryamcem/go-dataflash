package dataflash

import (
	"reflect"
	"testing"
)

func TestDecodeMessageBody_SingleUint8(t *testing.T) {
	schema := &Schema{
		Format:  "B",
		Columns: "Value",
		Length:  4, // 3-byte header + 1 byte data
	}

	body := []byte{123}

	result, err := DecodeMessageBody(body, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]any{
		"Value": uint8(123),
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestDecodeMessageBody_MultipleUnsignedIntegers(t *testing.T) {
	schema := &Schema{
		Format:  "BHI",
		Columns: "Field1,Field2,Field3",
		Length:  10, // 3 + 1 + 2 + 4
	}

	// B: 10
	// H: 1000 (0x03E8) = E8 03 in little-endian
	// I: 123456 (0x0001E240) = 40 E2 01 00 in little-endian
	body := []byte{
		10,                   // B
		0xE8, 0x03,           // H
		0x40, 0xE2, 0x01, 0x00, // I
	}

	result, err := DecodeMessageBody(body, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]any{
		"Field1": uint8(10),
		"Field2": uint16(1000),
		"Field3": uint32(123456),
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestDecodeMessageBody_SignedIntegers(t *testing.T) {
	schema := &Schema{
		Format:  "bhi",
		Columns: "Int8,Int16,Int32",
		Length:  10, // 3 + 1 + 2 + 4
	}

	// b: -42 (0xD6)
	// h: -1000 (0xFC18) = 18 FC in little-endian
	// i: -123456 (0xFFFE1DC0) = C0 1D FE FF in little-endian
	body := []byte{
		0xD6,                   // b: -42
		0x18, 0xFC,             // h: -1000
		0xC0, 0x1D, 0xFE, 0xFF, // i: -123456
	}

	result, err := DecodeMessageBody(body, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]any{
		"Int8":  int8(-42),
		"Int16": int16(-1000),
		"Int32": int32(-123456),
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestDecodeMessageBody_ScaledValues(t *testing.T) {
	schema := &Schema{
		Format:  "cL",
		Columns: "Altitude,Latitude",
		Length:  9, // 3 + 2 + 4
	}

	// c: 10050 (100.50 * 100) = 0x273A = 3A 27 in little-endian
	// L: 377487360 (37.7487360 * 1e7) = 0x168378C0 = C0 78 83 16 in little-endian
	body := []byte{
		0x3A, 0x27,             // c: altitude scaled
		0xC0, 0x78, 0x83, 0x16, // L: latitude
	}

	result, err := DecodeMessageBody(body, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check altitude (scaled by 0.01)
	alt, ok := result["Altitude"].(float64)
	if !ok {
		t.Fatalf("Altitude is not float64")
	}
	if alt < 100.41 || alt > 100.43 {
		t.Errorf("Altitude: got %v, want ~100.42", alt)
	}

	// Check latitude (scaled by 1e-7)
	lat, ok := result["Latitude"].(float64)
	if !ok {
		t.Fatalf("Latitude is not float64")
	}
	if lat < 37.77 || lat > 37.78 {
		t.Errorf("Latitude: got %v, want ~37.771", lat)
	}
}

func TestDecodeMessageBody_String(t *testing.T) {
	schema := &Schema{
		Format:  "n",
		Columns: "Name",
		Length:  7, // 3 + 4
	}

	// "GPS" with null terminator
	body := []byte{'G', 'P', 'S', 0x00}

	result, err := DecodeMessageBody(body, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]any{
		"Name": "GPS",
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestDecodeMessageBody_NullTerminator(t *testing.T) {
	// Format string with null terminator should stop parsing
	schema := &Schema{
		Format:  "BH\x00XX", // Null after H
		Columns: "Field1,Field2,Field3,Field4",
		Length:  6, // 3 + 1 + 2
	}

	body := []byte{
		42,         // B
		0x10, 0x27, // H
		// No more data after null terminator
	}

	result, err := DecodeMessageBody(body, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only parse Field1 and Field2
	if len(result) != 2 {
		t.Errorf("expected 2 fields, got %d", len(result))
	}

	if _, ok := result["Field1"]; !ok {
		t.Error("Field1 missing")
	}
	if _, ok := result["Field2"]; !ok {
		t.Error("Field2 missing")
	}
	if _, ok := result["Field3"]; ok {
		t.Error("Field3 should not be parsed")
	}
}

func TestDecodeMessageBody_MoreFormatThanColumns(t *testing.T) {
	// More format characters than column names
	schema := &Schema{
		Format:  "BHI",
		Columns: "Field1,Field2", // Only 2 columns but 3 format chars
		Length:  10,
	}

	body := []byte{
		10,
		0xE8, 0x03,
		0x40, 0xE2, 0x01, 0x00,
	}

	result, err := DecodeMessageBody(body, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only parse 2 fields
	if len(result) != 2 {
		t.Errorf("expected 2 fields, got %d", len(result))
	}
}
