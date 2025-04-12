; --- Config ---
.setcpu "6502"
.include "zeropage.inc"      ; Optional

; --- Constants ---
ARRAY_START = $0200
ARRAY_LEN   = 4             ; Number of elements in the array
RESULT_ADDR = $0002         ; Zero-page address to store the 8-bit sum

; --- Memory Segments ---

; == ZEROPAGE ==
.segment "ZEROPAGE"
.org $00
    result: .res 1           ; Reserve 1 byte for result

; == DATA ==
.segment "DATA"
.org ARRAY_START
    array_data: .byte $0A, $14, $1E, $05   ; 10, 20, 30, 5 (decimal)

; == CODE ==
.segment "CODE"
.org $0600
start:                       ; Start label is defined here at $0600
    LDY #0              ; Initialize Y index register to 0
    LDA #0              ; Initialize Accumulator (sum) to 0
    CLC                 ; Clear Carry flag before starting addition loop

loop:
    ADC array_data,Y    ; Absolute,Y addressing mode
    INY                 ; Increment Y index
    CPY #ARRAY_LEN      ; Compare Y to the number of elements
    BNE loop            ; Branch back to loop if Y != ARRAY_LEN (Z flag clear)

    STA RESULT_ADDR     ; Store the final 8-bit sum in $0002

done:
    BRK                 ; Halt execution (jumps to IRQ vector)

; ===============================
; == VECTORS (Defined LAST) ==
; ===============================
.segment "VECTORS"
.org $FFFA
    ; Now 'start' should be resolvable as it was defined earlier in the file
    .addr start             ; NMI Vector ($FFFA/$FFFB) - Point to start ($0600)
    .addr start             ; Reset Vector ($FFFC/$FFFD) - Point to start ($0600)
    .addr start             ; IRQ/BRK Vector ($FFFE/$FFFF) - Point to start ($0600)