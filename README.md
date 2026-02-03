# go-dataflash

ArduPilot DataFlash log parser written in Go.

## !!!Work in Progress

This project is currently in early development and was created for fun and learning purposes. The code is raw and not ready for use. Expect breaking changes, incomplete features, and rough edges.

## About

go-dataflash is a parser for ArduPilot DataFlash binary logs (`.bin` files). It reads flight telemetry data from ArduPilot-based flight controllers.

## Current Status

### v1.0.0
- [x] Two-pass parsing architecture
- [x] FMT (format) message parsing
- [x] Message schema discovery
- [x] Data message parsing
- [x] Message filtering

### v1.1.0
- [x] Message tracking (LineNo, TimeUS)
- [x] Log slicing by line number or time
- [x] Units and multipliers support (FMTU)

### v2.0.0
- [x] Performance improvements (~15-40 times faster with bufio)

See [TODO](https://github.com/pryamcem/go-dataflash/tree/master/TODO.md)

## Usage

See [examples/parse_log](https://github.com/pryamcem/go-dataflash/tree/master/examples/parse_log) for a complete working example.

### Basic Usage

```go
import "github.com/pryamcem/go-dataflash"

parser, _ := dataflash.NewParser("log.bin")
defer parser.Close()

for {
    msg, err := parser.ReadMessage()
    if err == io.EOF {
        break
    }
    // Process msg.Name and msg.Fields
}
```

### Filtering Messages

```go
parser.SetFilter("GPS", "IMU")  // Only parse GPS and IMU messages

for {
    msg, err := parser.ReadMessage()
    if err == io.EOF {
        break
    }
    // msg.Name will be either "GPS" or "IMU"
}
```

### Units and Scaled Values

Fields are automatically scaled based on their format character and FMTU multipliers:

```go
msg, _ := parser.ReadMessage()

// Fields are already scaled during parsing
// - Format characters like 'c', 'e', 'L' include built-in scaling
// - FMTU multipliers are applied for other formats (e.g., 'Q', 'I')
rawTimeUS := msg.Fields["TimeUS"]  // uint64 value

// Get scaled value with unit (returns any type)
scaled, unit, _ := msg.GetScaled("TimeUS")  // float64(44.167), "s" (F mult: /1e6)

// GPS altitude uses format 'e' (int32*100) - already scaled, type preserved
alt, unit, _ := msg.GetScaled("Alt")  // float64(275.3), "m" (no mult applied)

// Status field with no multiplier - original type preserved
status, _, _ := msg.GetScaled("Status")  // uint8(3) (original type)

// Get all fields with units (types preserved when no scaling needed)
scaledFields := msg.GetScaledFields()
for name, sv := range scaledFields {
    if sv.Unit != "" {
        fmt.Printf("%s: %v %s\n", name, sv.Value, sv.Unit)
    }
}
```

## Super fast
Parsing 48Mb log with 1,109,301 messages for 1.7s and parsing only GPS and IMU for 0.45s.
[See benchmark](https://github.com/pryamcem/go-dataflash/tree/master/benchmark_test.go) and try on your logs.
```
goos: linux
goarch: amd64
pkg: github.com/pryamcem/go-dataflash
cpu: 11th Gen Intel(R) Core(TM) i7-1185G7 @ 3.00GHz
BenchmarkParseAllMessages-8   	       1	1567094441 ns/op	1644086928 B/op	15049412 allocs/op
BenchmarkParseAllMessages-8   	       1	1560083874 ns/op	1644071344 B/op	15049393 allocs/op
BenchmarkParseAllMessages-8   	       1	1874030880 ns/op	1644079008 B/op	15049424 allocs/op
BenchmarkParseFiltered-8      	       3	 446275414 ns/op	67733824 B/op	  515493 allocs/op
BenchmarkParseFiltered-8      	       3	 462084433 ns/op	67733792 B/op	  515493 allocs/op
BenchmarkParseFiltered-8      	       3	 450877264 ns/op	67733733 B/op	  515492 allocs/op
PASS
ok  	github.com/pryamcem/go-dataflash	9.974s
```

## DataFlash Format Overview

### Structure
- Each message starts with a 3-byte header: `0xA3`, `0x95`, `msgType`
- First messages are FMT (Format) messages (type 128) that define all other message types
- Data messages follow, using the formats defined by FMT messages

### FMT Message Structure
- Type: uint8 (the message type this format describes)
- Length: uint8 (total message length including 3-byte header)
- Name: 4-char string (message name, e.g., "GPS", "IMU")
- Format: 16-char string (format specifiers: `B`=uint8, `h`=int16, `H`=uint16, `i`=int32, `I`=uint32, `f`=float, `d`=double, `n`=char[4], `N`=char[16], `Z`=char[64], `c`=int16*100, `C`=uint16*100, etc.)
- Columns: 64-char string (comma-separated column names)

### Key Implementation Notes
1. Build a map of `msgType -> FMT` as you read FMT messages
2. All strings are null-terminated but have fixed max lengths
3. Binary encoding is little-endian
4. The format string tells how to decode each field in order

## Learning Goals

This project was created to:
- Remember Go
- Understand binary file formats
- Practice parsing techniques
- Explore ArduPilot telemetry data

## References

- [ArduPilot DataFlash Log Format](https://ardupilot.org/dev/docs/loganalysis.html)
- [pymavlink](https://github.com/ArduPilot/pymavlink) - Python reference implementation
