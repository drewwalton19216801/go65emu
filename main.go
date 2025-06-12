package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	cpu6502 "github.com/drewwalton19216801/sixty502" // Assuming cpu.go is in cpu6502 directory

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 800
	screenHeight = 600

	charWidth     = 10 // Estimated width for default font or loaded font
	charHeight    = 20 // Estimated height
	memViewX      = 350
	memViewY      = 20
	memViewCols   = 16
	memViewRows   = 25 // How many rows of memory to display
	codeViewLines = 15 // <<<--- ADD THIS CONSTANT
)

// Bus represents the 64KB memory space
type Bus struct {
	mem [65536]byte
}

// Read implements the cpu6502.Bus interface
func (r *Bus) Read(addr uint16) uint8 {
	// No boundary check needed due to uint16, wraps around naturally
	return r.mem[addr]
}

// Write implements the cpu6502.Bus interface
func (r *Bus) Write(addr uint16, data uint8) {
	r.mem[addr] = data
}

// LoadProgram reads a binary file into memory at a specific address
func (r *Bus) LoadProgram(filename string, startAddr uint16) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	log.Printf("Loading %d bytes from %s to address $%04X", len(data), filename, startAddr)

	for i, b := range data {
		addr := startAddr + uint16(i)
		if addr < startAddr { // Check for wrap around 64k boundary
			log.Printf("Warning: Program too large, wrapping around memory at address $%04X", addr)
			// Allow wrap around for now, could also return error
		}
		r.Write(addr, b)
	}
	return nil
}

func main() {
	// Command line flags
	binFile := flag.String("bin", "", "Path to the .bin program file to load.")
	loadAddr := flag.Int("addr", 0x0200, "Memory address (decimal) to load the .bin file (default: 512 / $0200). Use 0x prefix for hex.") // Common starting point, avoid zero page/stack

	flag.Parse()

	// Initialize Memory and CPU
	ram := &Bus{} // Zero-initialized memory
	cpu := cpu6502.NewCPU(ram)

	// Load program if specified
	var entryPoint uint16 = 0xFFFC // Default reset vector if no program loaded
	if *binFile != "" {
		loadAddress := uint16(*loadAddr)
		err := ram.LoadProgram(*binFile, loadAddress)
		if err != nil {
			log.Fatalf("Error loading program: %v", err)
		}
		entryPoint = loadAddress // Set PC to start of loaded program after reset
		log.Printf("Program loaded. Resetting PC to entry point $%04X.", entryPoint)
	} else {
		log.Println("No .bin file specified. Loading default program at reset vector.")
		// Optional: you could put a minimal default program (e.g., infinite loop JMP) at the reset vector location
		ram.Write(0xFFFC, 0x00)
		ram.Write(0xFFFD, 0x80) // Example: reset vector points to $8000
		ram.Write(0x8000, 0x4C) // JMP
		ram.Write(0x8001, 0x00) // $8000
		ram.Write(0x8002, 0x80)
		entryPoint = 0x8000 // If using above example
	}

	// --- Perform initial CPU Reset ---
	cpu.Reset()
	// If a program was loaded, override the PC set by Reset() to point to the program start
	if *binFile != "" {
		cpu.PC = entryPoint
	}
	// --- End Reset ---

	// Raylib Initialization
	rl.InitWindow(screenWidth, screenHeight, "Go 6502 Emulator")
	defer rl.CloseWindow()

	// Try loading a better font (adjust path if needed)
	// You might need to download a font like "Source Code Pro" or "Fira Code"
	fontPath := "font/source-code-pro/SourceCodePro-Regular.otf" // Example path
	var font rl.Font
	if _, err := os.Stat(fontPath); err == nil {
		font = rl.LoadFontEx(fontPath, int32(charHeight-4), nil, 0) // Load with specific size
	} else {
		log.Printf("Warning: Font not found at %s. Using default font.", fontPath)
		font = rl.GetFontDefault() // Fallback to default
	}
	defer rl.UnloadFont(font) // Unload font when done

	rl.SetTargetFPS(60)

	// Emulator State
	running := false
	memViewStartAddress := uint16(0x0000) // Start memory view at address 0

	// Disassembly Cache (simple approach)
	disassembly := make(map[uint16]string)
	disassemblyNeedsUpdate := true // Flag to regenerate disassembly

	for !rl.WindowShouldClose() {
		// --- Input Handling ---
		if rl.IsKeyPressed(rl.KeySpace) {
			running = !running
		}
		if rl.IsKeyPressed(rl.KeyS) {
			if !running {
				// Execute one full instruction
				for cpu.Cycles > 0 {
					cpu.Clock()
				}
				// Now execute the *next* instruction completely
				cpu.Clock() // Fetch opcode
				for cpu.Cycles > 0 {
					cpu.Clock() // Execute remaining cycles
				}
				disassemblyNeedsUpdate = true // PC changed
			}
		}
		if rl.IsKeyPressed(rl.KeyR) {
			cpu.Reset()
			if *binFile != "" {
				cpu.PC = entryPoint // Point back to loaded program start
			}
			running = false
			disassemblyNeedsUpdate = true
			log.Println("CPU Reset.")
		}
		if rl.IsKeyPressed(rl.KeyPageDown) {
			memViewStartAddress += uint16(memViewRows * memViewCols)
			// Wrap around check (optional, uint16 handles it)
			if memViewStartAddress < uint16(memViewRows*memViewCols) && memViewStartAddress != 0 {
				memViewStartAddress = 0xFFFF - uint16(memViewRows*memViewCols) + 1
			}
		}
		if rl.IsKeyPressed(rl.KeyPageUp) {
			// Prevent underflow with uint16 arithmetic
			offset := uint16(memViewRows * memViewCols)
			if memViewStartAddress >= offset {
				memViewStartAddress -= offset
			} else {
				// Wrap around to the top
				memViewStartAddress = 0xFFFF - uint16(memViewRows*memViewCols) + 1
			}
		}
		// Load file functionality - More complex with GUI, using CLI for now
		if rl.IsKeyPressed(rl.KeyL) {
			// Placeholder: Ideally, this would open a file dialog
			log.Println("Load function (L key) not implemented in GUI. Use -bin flag on start.")
		}

		// --- Update Emulator ---
		if running {
			// Execute a batch of cycles per frame to keep it responsive
			// Adjust cyclesPerFrame based on performance/desired speed
			const cyclesPerFrame = 10000 // Example value
			frameCycles := uint64(0)
			for frameCycles < cyclesPerFrame {
				if cpu.Cycles == 0 {
					cpu.Clock() // Fetch next opcode, decrement cycles
					frameCycles++
					disassemblyNeedsUpdate = true // PC changed
				} else {
					cpu.Clock() // Execute cycle of current instruction
					frameCycles++
				}
			}
		}

		// --- Update Disassembly (if needed) ---
		if disassemblyNeedsUpdate {
			// Disassemble a small region around PC
			startDisAddr := cpu.PC
			if startDisAddr > uint16(codeViewLines/2) {
				startDisAddr -= uint16(codeViewLines / 2)
			} else {
				startDisAddr = 0
			}
			endDisAddr := startDisAddr + uint16(codeViewLines*3) // Estimate bytes needed
			if endDisAddr < startDisAddr {
				endDisAddr = 0xFFFF // Prevent wrap
			}
			disassembly = cpu.Disassemble(startDisAddr, endDisAddr)
			disassemblyNeedsUpdate = false // Reset flag
		}

		// --- Drawing ---
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		// Draw UI Sections
		DrawCPUState(cpu, 10, 20, font)
		DrawControls(10, 180, font, running)
		DrawCodeView(cpu, disassembly, 10, 350, codeViewLines, font)
		DrawMemoryView(ram, memViewStartAddress, memViewX, memViewY, memViewRows, memViewCols, font)

		rl.EndDrawing()
	}
}

// DrawCPUState displays the CPU registers and flags
func DrawCPUState(cpu *cpu6502.CPU, x, y int32, font rl.Font) {
	statusText := cpu6502.FormatFlags(cpu.P)
	stateLines := []string{
		fmt.Sprintf("PC: $%04X", cpu.PC),
		fmt.Sprintf(" A: $%02X [%d]", cpu.A, cpu.A),
		fmt.Sprintf(" X: $%02X [%d]", cpu.X, cpu.X),
		fmt.Sprintf(" Y: $%02X [%d]", cpu.Y, cpu.Y),
		fmt.Sprintf("SP: $%04X", 0x0100+uint16(cpu.SP)), // Show full stack address
		fmt.Sprintf(" P: $%02X [%s]", uint8(cpu.P), statusText),
		fmt.Sprintf("Cycles: %d", cpu.TotalCycles()),
	}

	lineHeight := int32(charHeight)
	for i, line := range stateLines {
		rl.DrawTextEx(font, line, rl.NewVector2(float32(x), float32(y+int32(i)*lineHeight)), float32(font.BaseSize), 1, rl.Black)
	}
}

// DrawControls displays the keyboard shortcuts
func DrawControls(x, y int32, font rl.Font, running bool) {
	status := "Paused"
	if running {
		status = "Running"
	}
	controlLines := []string{
		fmt.Sprintf("STATUS: %s", status),
		"[SPACE] = Run/Pause",
		"[S]     = Step Instruction",
		"[R]     = Reset CPU",
		"[PgUp]  = Scroll Mem Up",
		"[PgDn]  = Scroll Mem Down",
		"[L]     = Load (Use CLI -bin)",
	}
	lineHeight := int32(charHeight)
	for i, line := range controlLines {
		rl.DrawTextEx(font, line, rl.NewVector2(float32(x), float32(y+int32(i)*lineHeight)), float32(font.BaseSize), 1, rl.DarkGray)
	}
}

// DrawMemoryView displays a portion of RAM
func DrawMemoryView(bus *Bus, startAddr uint16, x, y, rows, cols int32, font rl.Font) {
	lineHeight := int32(charHeight)
	addr := startAddr
	asciiStr := make([]byte, cols)

	for r := int32(0); r < rows; r++ {
		// Draw Address
		addrHex := fmt.Sprintf("$%04X: ", addr)
		rl.DrawTextEx(font, addrHex, rl.NewVector2(float32(x), float32(y+r*lineHeight)), float32(font.BaseSize), 1, rl.Blue)

		// Draw Hex Bytes & Build ASCII string
		hexStr := ""
		for c := int32(0); c < cols; c++ {
			val := bus.Read(addr + uint16(c))
			hexStr += fmt.Sprintf("%02X ", val)
			if val >= 32 && val <= 126 { // Printable ASCII
				asciiStr[c] = val
			} else {
				asciiStr[c] = '.'
			}
		}
		// Draw Hex String
		hexX := float32(x + int32(len(addrHex))*charWidth/2 + 10) // Adjust spacing based on font/char width
		rl.DrawTextEx(font, hexStr, rl.NewVector2(hexX, float32(y+r*lineHeight)), float32(font.BaseSize), 1, rl.Black)

		// Draw ASCII String
		asciiX := hexX + float32(len(hexStr))*charWidth/2 + 10 // Adjust spacing
		rl.DrawTextEx(font, string(asciiStr), rl.NewVector2(asciiX, float32(y+r*lineHeight)), float32(font.BaseSize), 1, rl.DarkGreen)

		addr += uint16(cols)  // Move to next line's address
		if addr < startAddr { // Stop if wrapped around (shouldn't happen with rows*cols << 64k)
			break
		}
	}
}

// DrawCodeView displays disassembled code around the PC
func DrawCodeView(cpu *cpu6502.CPU, disassembly map[uint16]string, x, y, linesToShow int32, font rl.Font) {
	lineHeight := int32(charHeight)
	currentLine := int32(0)
	foundPC := false

	// Try to center the view around PC by iterating backwards first
	lookBehind := uint16(linesToShow / 2)
	startAddr := cpu.PC
	if startAddr > lookBehind {
		startAddr -= lookBehind
	} else {
		startAddr = 0
	}

	// Iterate through potential addresses to display
	tempAddr := startAddr
	for currentLine < linesToShow {
		line, exists := disassembly[tempAddr]
		if !exists {
			// If the start address wasn't in the map, try finding the next valid one
			// This is imperfect but better than showing nothing.
			nextValidAddr := tempAddr + 1
			for ; nextValidAddr != tempAddr; nextValidAddr++ { // Search forward (with wrap)
				if _, ok := disassembly[nextValidAddr]; ok {
					break
				}
				if nextValidAddr == 0xFFFF {
					break
				} // Avoid infinite loop on empty map segment
			}
			if nextValidAddr == tempAddr { // No more valid instructions found in range
				break
			}
			tempAddr = nextValidAddr
			line = disassembly[tempAddr]
		}

		// Determine the number of bytes this instruction uses (approximate from disassembly)
		bytesUsed := 1 // At least the opcode
		opAddr := tempAddr
		nextAddr := tempAddr + 1
		// This is a simple heuristic, might be wrong for some addr modes/ops
		if len(line) > 5 { // Basic check if there's likely an operand
			peekAddr1 := tempAddr + 1
			peekAddr2 := tempAddr + 2
			if _, exists1 := disassembly[peekAddr1]; !exists1 {
				bytesUsed++
				nextAddr++
				if _, exists2 := disassembly[peekAddr2]; !exists2 {
					bytesUsed++
					nextAddr++
				}
			}
		}

		color := rl.Black
		prefix := "  "
		if tempAddr == cpu.PC {
			color = rl.Red
			prefix = "->" // Indicate current instruction
			foundPC = true
		}

		displayText := fmt.Sprintf("%s%04X: %s", prefix, opAddr, line)
		rl.DrawTextEx(font, displayText, rl.NewVector2(float32(x), float32(y+currentLine*lineHeight)), float32(font.BaseSize), 1, color)
		currentLine++

		// Advance address to the next instruction based on our heuristic
		tempAddr = nextAddr
		if tempAddr < opAddr { // Stop if wrapped
			break
		}
	}

	// If PC wasn't found (e.g., PC points to data), show a basic line for it
	if !foundPC {
		line, exists := disassembly[cpu.PC]
		if !exists {
			line = fmt.Sprintf("%s ???", cpu6502.Instruction{Name: "???"}.Name) // Show something
		}
		displayText := fmt.Sprintf("->%04X: %s", cpu.PC, line)
		rl.DrawTextEx(font, displayText, rl.NewVector2(float32(x), float32(y+currentLine*lineHeight)), float32(font.BaseSize), 1, rl.Red)
	}
}
