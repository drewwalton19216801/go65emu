# go65emu
A simple 6502-based emulator written in Go

## Examples

### Sum Array

`go run main.go -bin examples/bin/sum_array.bin -addr 0x0000`

Sums a small array of numbers stored in memory. Features used: *ABSY* addressing, *CPY* comparisons, *BNE* branching, *ADC* arithmetic.

## Screenshot

![Go 6502 emulator state view](https://raw.githubusercontent.com/drewwalton19216801/go65emu/refs/heads/dev/screenshots/sum-array.png)