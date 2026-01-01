# go-dataflash

ArduPilot DataFlash log parser written in Go.

## !!!Work in Progress

This project is currently in early development and was created for fun and learning purposes. The code is raw and not ready for use. Expect breaking changes, incomplete features, and rough edges.

## About

go-dataflash is a parser for ArduPilot DataFlash binary logs (`.bin` files). It reads flight telemetry data from ArduPilot-based flight controllers.

## Current Status

- [x] Two-pass parsing architecture (very slow and ineffective)
- [x] FMT (format) message parsing
- [x] Message schema discovery
- [x] Data message parsing
- [x] Message filtering
- [  ] Message statistics
- [  ] Metadata extraction
- [  ] Performance improvements

See [TODO](https://github.com/pryamcem/go-dataflash/tree/master/TODO.md)

## Usage

See [examples/parse_log](https://github.com/pryamcem/go-dataflash/tree/master/examples/parse_log) for a complete working example.

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
