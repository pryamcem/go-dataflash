# go-dataflash TODO

## v2.0 - Nice-to-Have Features

### Message Statistics
- [ ] Add `Stats` struct with message counts, time ranges, duration
- [ ] Add tests for stats calculation
- [ ] Implement `GetStats() Stats` method
- [ ] Track stats during parsing (minimal overhead)

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

## v2.x - Metadata Extraction (Medium effort)

### MSG Message Parsing
- [ ] Add `Metadata` struct (Platform, Version, Commit, Hardware, etc.)
- [ ] Add tests for metadata extraction
- [ ] Extract platform from first MSG (ArduPlane, ArduCopter, etc.)
- [ ] Extract version from MSG (e.g., V4.6.3)
- [ ] Extract git commit hash from MSG (e.g., 3fc7011a)
- [ ] Extract hardware info from MSG messages
- [ ] Add `GetMetadata() Metadata` method

### Message Type List
- [ ] Build list of available message types during schema building
- [ ] Add `GetAvailableTypes() []string` method
- [ ] Sort types alphabetically for better UX
- [ ] Add to metadata struct

## v3.0 - Performance Improvements (High effort)

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

## Future Ideas (v4.0+)

- [ ] Support for TLOG format (telemetry logs)
- [ ] Message indexing for fast random access
- [ ] CSV export functionality
- [ ] Streaming API for real-time log processing
- [ ] Unit/multiplier support (FMTU messages)
- [ ] Schema validation and version checking
