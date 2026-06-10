package dataflash

import (
	"bytes"
	"testing"
)

// fmtRecord builds a FMT (type 128) record defining a schema.
// Layout: HEAD1, HEAD2, 128, Type, Length, Name[4], Format[16], Columns[64].
func fmtRecord(defType, length uint8, name, format, columns string) []byte {
	rec := []byte{HEAD1, HEAD2, FMTType, defType, length}
	for _, f := range []struct {
		s string
		n int
	}{{name, 4}, {format, 16}, {columns, 64}} {
		field := make([]byte, f.n)
		copy(field, f.s)
		rec = append(rec, field...)
	}
	return rec
}

// Regression: a known non-FMT body containing HEAD1/HEAD2 magic must not desync
// buildSchemas and drop the FMT records that follow it.
func TestBuildSchemasMagicInBody(t *testing.T) {
	const (
		fooType  = 10
		baroType = 11
	)

	// FOO body opens with magic + the FMT type byte (0x80): a byte-scan
	// re-aligns here and eats the BARO definition that follows.
	fooBody := make([]byte, 20)
	fooBody[0] = HEAD1
	fooBody[1] = HEAD2
	fooBody[2] = FMTType
	fooLen := uint8(HeaderSize + len(fooBody))

	var log []byte
	log = append(log, fmtRecord(fooType, fooLen, "FOO", "f", "Val")...)
	log = append(log, HEAD1, HEAD2, fooType)
	log = append(log, fooBody...)
	log = append(log, fmtRecord(baroType, 11, "BARO", "If", "TimeUS,Alt")...)

	parser, err := NewParser(bytes.NewReader(log))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer parser.Close()

	schemas := parser.GetSchemas()

	if foo, ok := schemas[fooType]; !ok || foo.Name != "FOO" {
		t.Fatalf("FOO schema missing or wrong: %+v", schemas[fooType])
	}

	baro, ok := schemas[baroType]
	if !ok {
		t.Fatal("BARO schema was dropped: parser desynced on magic bytes in FOO body")
	}
	if baro.Name != "BARO" {
		t.Errorf("expected BARO, got %q", baro.Name)
	}
}
