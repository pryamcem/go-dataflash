# go-dataflash TODO

## v1.x - API Improvements

### ~~Message Statistics~~ (SKIPPED)
**Why skipped**: Users can easily calculate basic stats (message counts, rates, duration) themselves by iterating through messages. Complex stats (sample rate consistency, gap analysis) add too much complexity for minimal benefit. Keep the library focused on parsing, not analysis.

### Iterator Pattern (Optional)
- [ ] Add tests for iterator pattern
- [ ] Implement `Messages() <-chan *Message` for range iteration
- [ ] Handle errors via separate error channel or in Message
- [ ] Add context support for cancellation
- [ ] Update examples to show iterator usage

### Field Access Helpers (Optional)
- [ ] Add tests for all getter methods
- [ ] Add `Message.GetInt64(field string) (int64, error)` 
- [ ] Add `Message.GetFloat64(field string) (float64, error)`
- [ ] Add `Message.GetString(field string) (string, error)`
- [ ] Add `Message.GetBool(field string) (bool, error)`
- [ ] Add convenience methods for common fields (TimeUS, Lat, Lng, Alt)

## ~~Metadata Extraction~~ (SKIPPED)
**Why skipped**: 
- No standard format across firmwares (ArduPilot, PX4, iNav, etc.)
- Requires brittle pattern matching that breaks with firmware updates
- Not all logs have MSG messages with metadata
- Users can easily filter MSG messages themselves: `parser.SetFilter("MSG")`
- Adds maintenance burden with little value
- Keep library simple: parse data reliably, let users interpret metadata

## v1.1 - Units and Multipliers Support ✓ COMPLETED

**Approach**: Keep raw values in `Fields`, add scaled values via methods (backward compatible)

- [x] Parse FMTU messages and store units/mults in Schema
- [x] Create unit/multiplier lookup maps (s→"seconds", F→1e-6, etc.)
- [x] Add `ScaledValue` type with Value and Unit fields
- [x] Add `msg.GetScaled(field)` → (value, unit, error)
- [x] Add `msg.GetScaledFields()` → map of all scaled values
- [x] Write tests and update docs
- [x] Create units_example demonstrating scaling usage

**Why useful**: Automatic unit conversion, proper data interpretation, backward compatible
**Implementation**: Added units.go, message.go with GetScaled methods, updated parser to parse FMTU messages during schema building phase

## v2.x - Performance Improvements (High Priority)

### Buffered Reader
- [ ] Replace direct file I/O with `bufio.Reader`
- [ ] Add `NewParserWithBuffer(filename, bufferSize)` constructor
- [ ] Benchmark before/after to measure improvement
- [ ] Tune buffer size (test 64KB, 128KB, 256KB)
- [ ] Update docs with performance notes

### Parallel Parsing (Optional)
- [ ] Add `ReadAllMessages() ([]*Message, error)` method
- [ ] Implement goroutine pool for parallel message decoding
- [ ] Add benchmark for large files
- [ ] Consider memory vs speed tradeoffs
- [ ] Make configurable (workers count)

## Future Ideas (v3.0+)

- [ ] Support for TLOG format (telemetry logs)
- [ ] Message indexing for fast random access
- [ ] CSV export functionality
- [ ] Streaming API for real-time log processing
- [ ] Schema validation and version checking
