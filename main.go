package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	// Import your library
	cpu6502 "github.com/drewwalton19216801/sixty502"
)

// RamBus implements the cpu6502.Bus interface using a simple byte slice for RAM.
type RamBus struct {
	mem []uint8
}

// NewRamBus creates a 64KB RAM bus.
func NewRamBus() *RamBus {
	return &RamBus{
		mem: make([]uint8, 65536), // 64KB address space
	}
}

// Read implements the Bus interface with corrected bounds check.
func (r *RamBus) Read(addr uint16) uint8 {
	// CORRECTED CHECK: Compare addr (promoted to int) with int length.
	if int(addr) >= len(r.mem) {
		// Log with more info: max valid address is len-1
		log.Printf("WARN: Read access out of bounds: $%04X (max $%04X)", addr, len(r.mem)-1)
		return 0 // Or handle differently, e.g., open bus behavior
	}
	return r.mem[addr]
}

// Write implements the Bus interface with corrected bounds check.
func (r *RamBus) Write(addr uint16, data uint8) {
	// CORRECTED CHECK: Compare addr (promoted to int) with int length.
	if int(addr) >= len(r.mem) {
		// Log with more info: max valid address is len-1
		log.Printf("WARN: Write access out of bounds: $%04X = %02X (max $%04X)", addr, data, len(r.mem)-1)
		return // Prevent writing out of bounds
	}
	r.mem[addr] = data
}

// LoadProgram reads a binary file and copies its content into RAM.
func (r *RamBus) LoadProgram(filename string, startAddr uint16) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read program file '%s': %w", filename, err)
	}

	// Check if the *loaded data itself* would exceed memory limits from startAddr
	if int(startAddr)+len(data) > len(r.mem) {
		// Note: If data is 65536 and startAddr is 0, this check passes (correctly).
		return fmt.Errorf("program file '%s' (%d bytes) is too large to load at $%04X (max addr $%04X)", filename, len(data), startAddr, len(r.mem)-1)
	}

	copy(r.mem[startAddr:], data)
	log.Printf("Loaded %d bytes from '%s' starting at address $%04X", len(data), filename, startAddr)
	return nil
}

func main() {
	// --- Command Line Arguments ---
	programPath := flag.String("bin", "", "Path to the 6502 binary program file to load.")
	maxSteps := flag.Int("steps", 100, "Maximum number of instructions to execute.")
	loadAddrInput := flag.Uint("load", 0x0000, "Address to load the binary program into (decimal or 0xHEX).")

	flag.Parse()

	if *programPath == "" {
		log.Fatal("Error: Program path must be provided using -bin flag.")
	}
	if *loadAddrInput > 0xFFFF {
		log.Fatal("Error: Invalid load address. Must be between 0x0000 and 0xFFFF.")
	}
	loadAddr := uint16(*loadAddrInput)

	// --- Setup ---
	ram := NewRamBus()
	cpu := cpu6502.NewCPU(ram)

	// --- Load Program ---
	err := ram.LoadProgram(*programPath, loadAddr)
	if err != nil {
		log.Fatalf("Error loading program: %v", err)
	}

	// --- Reset CPU ---
	cpu.Reset()

	fmt.Println("--- CPU State After Reset ---")
	initialCycles := cpu.Cycles // Use accessor if available
	// Use accessor in loop if available
	for cpu.Cycles > 0 {
		cpu.Clock()
	}
	fmt.Printf("%s (Reset took %d cycles)\n", cpu.GetState(), initialCycles)
	fmt.Println("-----------------------------")

	// --- Execution Loop ---
	fmt.Println("--- Executing Instructions ---")

	var step int // Declare outer step

	// Loop until execution halts
	for step = 0; step < *maxSteps; step++ {

		prevState := cpu.GetState()
		prevPC := cpu.PC

		cpu.Clock()
		// Use accessor in loop if available
		for cpu.Cycles > 0 {
			cpu.Clock()
		}

		fmt.Printf("Step %d: [%04X] %s\n", step+1, prevPC, prevState)
		fmt.Printf("        -> %s\n", cpu.GetState())
		fmt.Println("-----------------------------")

		// Halt detection logic
		currentOpcode := cpu.Opcode()
		currentPC := cpu.PC

		if currentOpcode == 0x00 {
			brkVector := uint16(ram.Read(0xFFFE)) | (uint16(ram.Read(0xFFFF)) << 8)
			if currentPC == brkVector {
				fmt.Println("--- BRK executed, PC matches IRQ vector. Assuming intended HALT. ---")
				break
			} else {
				fmt.Println("--- BRK executed. Execution continues from IRQ vector. ---")
			}
		} else if currentPC == prevPC {
			fmt.Println("--- Program Counter hasn't changed, assuming infinite loop/halt. ---")
			break
		}
	} // --- End of for loop ---

	// Final status message
	if step < *maxSteps {
		fmt.Printf("--- Execution halted or finished at step %d ---\n", step+1)
	} else if *maxSteps > 0 {
		fmt.Printf("--- Execution stopped after %d steps ---\n", *maxSteps)
	} else {
		fmt.Println("--- Execution finished (max steps was 0) ---")
	}
}
