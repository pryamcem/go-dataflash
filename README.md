# go-dataflash

ArduPilot DataFlash log parser written in Go.

## About

go-dataflash is a parser for ArduPilot DataFlash binary logs (`.bin` files). It reads flight telemetry data from ArduPilot-based flight controllers.

## Version History

### v3.0.0
- Lazy field decoding: messages keep their raw body and decode fields on demand
- `Message.Fields` is now a method, `Message.Fields()` (decodes all fields once and caches them)
- New `Message.Get(field)` to decode a single field without building a map
- New `Parser.ReadInto(msg)` to reuse a message and its buffer across reads
- Module path updated to `/v3`

Migrating from v2: replace `msg.Fields["X"]` with `msg.Get("X")` (one field) or `msg.Fields()["X"]` (many fields).

### v2.0.0
- Caller now owns the source — `NewParser` accepts `io.ReadSeeker`, `Close` is a no-op
- `ClearFilter` removed — use `SetFilter()` with no arguments instead
- `SliceType` changed from string to `int` enum for compile-time safety
- `GetScaled` now returns `ScaledValue` instead of `(any, string, error)`
- `GetSchemas` returns a copy to prevent external mutation
- Module path updated to `/v2`

### v1.3.0
- `NewParserFromSource` — parse from any `io.ReadSeeker`, not just files (by [@yur4uwe](https://github.com/yur4uwe))

### v1.2.0
- Performance improvements (~15-40 times faster with bufio)

### v1.1.0
- Message tracking (LineNo, TimeUS)
- Log slicing by line number or time
- Units and multipliers support (FMTU)

### v1.0.0
- Two-pass parsing architecture
- FMT (format) message parsing
- Message schema discovery
- Data message parsing
- Message filtering

## Usage

See [examples/parse_log](https://github.com/pryamcem/go-dataflash/tree/master/examples/parse_log) for a complete working example.

### Basic Usage

```go
import "github.com/pryamcem/go-dataflash/v3"

f, err := os.Open("log.bin")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

parser, err := dataflash.NewParser(f)
if err != nil {
    log.Fatal(err)
}

for {
    msg, err := parser.ReadMessage()
    if err == io.EOF || err == io.ErrUnexpectedEOF {
        break
    }
    // Process msg.Name, msg.TimeUS, and fields (see "Reading Fields")
}
```

### Reading Fields

Messages are decoded lazily: `ReadMessage` keeps the raw bytes and only decodes a field when you ask for it.

```go
// One or a few fields: decodes just that field
alt, ok := msg.Get("Alt")  // ok is false if the field does not exist

// Many fields: decodes all of them once and caches the result
for name, value := range msg.Fields() {
    fmt.Println(name, value)
}
```

If you need most of a message's fields, call `Fields()` once instead of calling `Get` for every column. `Get` per column is slower than `Fields()`.

### Reusing Messages

`ReadInto` fills a `Message` you provide and reuses its buffer, so reading does not allocate per message:

```go
var msg dataflash.Message
for {
    err := parser.ReadInto(&msg)
    if err == io.EOF || err == io.ErrUnexpectedEOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    // msg is only valid until the next ReadInto call
    if alt, ok := msg.Get("Alt"); ok {
        altitudes = append(altitudes, alt)  // decoded values are independent copies, safe to keep
    }
}
```

The message data is overwritten on the next call. Copy out anything you need to keep. Use `ReadMessage` if you want messages you can hold on to.

### Filtering Messages

```go
parser.SetFilter("GPS", "IMU")  // Only parse GPS and IMU messages

for {
    msg, err := parser.ReadMessage()
    if err == io.EOF || err == io.ErrUnexpectedEOF {
        break
    }
    // msg.Name will be either "GPS" or "IMU"
}

parser.SetFilter()  // Clear filter, all message types returned
```

### Units and Scaled Values

Fields are automatically scaled based on their format character and FMTU multipliers:

```go
msg, _ := parser.ReadMessage()

// Format characters like 'c', 'e', 'L' include built-in scaling when decoded.
// FMTU multipliers (e.g., for 'Q', 'I') are applied by GetScaled.
rawTimeUS, _ := msg.Get("TimeUS")  // uint64 value

// Get scaled value with unit
sv, _ := msg.GetScaled("TimeUS")  // sv.Value = float64(44.167), sv.Unit = "s"
sv, _ = msg.GetScaled("Alt")      // sv.Value = float64(275.3),  sv.Unit = "m"
sv, _ = msg.GetScaled("Status")   // sv.Value = uint8(3),        sv.Unit = ""

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
