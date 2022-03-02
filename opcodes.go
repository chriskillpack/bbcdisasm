package bbcdisasm

import (
	"fmt"
	"sort"
)

const labelFormatString = "label_%d"

// AddressingMode enumerates the different address modes of 6502 instructions
type AddressingMode int

// Addressing Modes
//  None        - no addressing mode                          - BRK
//  Accumulator - uses the accumulator register               - ASL A
//  Immediate   - using a data constant                       - LDA #FF
//  Absolute    - using a fixed address                       - LDA &1234
//  ZeroPage    - using a fixed zero page address             - LDA &12
//  ZeroPageX   - using zero page address+X                   - LDA &12,X
//  ZeroPageY   - using zero page address+Y (LDX only)        - LDX &12,Y
//  Indirect    - using an address stored in memory           - LDA (&1234)
//  AbsoluteX   - using an absolute address+X                 - LDA &1234,X
//  AbsoluteY   - using an absolute address+Y                 - LDA &1234,Y
//  IndirectX   - a table of zero page addresses indexed by X - LDA (&80,X)
//  IndirectY   - a table of zero page addresses indexed by Y - LDA (&80,Y)
const (
	None AddressingMode = iota
	Accumulator
	Immediate
	Absolute
	ZeroPage
	ZeroPageX
	ZeroPageY
	Indirect
	AbsoluteX
	AbsoluteY
	IndirectX
	IndirectY
)

type opcode struct {
	length   uint   // number of bytes include opcode
	name     string // human readable instruction 'name' of opcode
	addrMode AddressingMode
}

var (
	// OpCodesMap maps from first instruction opcode byte to 6502 instruction
	// Most opcodes from http://www.6502.org/tutorials/6502opcodes.html
	// ANC, SLO, SRE from https://github.com/mattgodbolt/jsbeeb/blob/master/6502.opcodes.js
	OpCodesMap = map[byte]opcode{
		0x69: {2, "ADC", Immediate},
		0x65: {2, "ADC", ZeroPage},
		0x75: {2, "ADC", ZeroPageX},
		0x6D: {3, "ADC", Absolute},
		0x7D: {3, "ADC", AbsoluteX},
		0x79: {3, "ADC", AbsoluteY},
		0x61: {2, "ADC", IndirectX},
		0x71: {2, "ADC", IndirectY},

		0x0B: {2, "ANC", Immediate},
		0x2B: {2, "ANC", Immediate},

		0x29: {2, "AND", Immediate},
		0x25: {2, "AND", ZeroPage},
		0x35: {2, "AND", ZeroPageX},
		0x2D: {3, "AND", Absolute},
		0x3D: {3, "AND", AbsoluteX},
		0x39: {3, "AND", AbsoluteY},
		0x21: {2, "AND", IndirectX},
		0x31: {2, "AND", IndirectY},

		0x0A: {1, "ASL", Accumulator},
		0x06: {2, "ASL", ZeroPage},
		0x16: {2, "ASL", ZeroPageX},
		0x0E: {3, "ASL", Absolute},
		0x1E: {3, "ASL", AbsoluteX},

		0x24: {2, "BIT", ZeroPage},
		0x2C: {3, "BIT", Absolute},

		0x10: {2, "BPL", None}, // all the branch instructions have special cased
		0x30: {2, "BMI", None}, // printing
		0x50: {2, "BVC", None},
		0x70: {2, "BVS", None},
		0x90: {2, "BCC", None},
		0xB0: {2, "BCS", None},
		0xD0: {2, "BNE", None},
		0xF0: {2, "BEQ", None},

		0x00: {1, "BRK", None},

		0xC9: {2, "CMP", Immediate},
		0xC5: {2, "CMP", ZeroPage},
		0xD5: {2, "CMP", ZeroPageX},
		0xCD: {3, "CMP", Absolute},
		0xDD: {3, "CMP", AbsoluteX},
		0xD9: {3, "CMP", AbsoluteY},
		0xC1: {2, "CMP", IndirectX},
		0xD1: {2, "CMP", IndirectY},

		0xE0: {2, "CPX", Immediate},
		0xE4: {2, "CPX", ZeroPage},
		0xEC: {3, "CPX", Absolute},

		0xC0: {2, "CPY", Immediate},
		0xC4: {2, "CPY", ZeroPage},
		0xCC: {3, "CPY", Absolute},

		0xC6: {2, "DEC", ZeroPage},
		0xD6: {2, "DEC", ZeroPageX},
		0xCE: {3, "DEC", Absolute},
		0xDE: {3, "DEC", AbsoluteX},

		0x49: {2, "EOR", Immediate},
		0x45: {2, "EOR", ZeroPage},
		0x55: {2, "EOR", ZeroPageX},
		0x4D: {3, "EOR", Absolute},
		0x5D: {3, "EOR", AbsoluteX},
		0x59: {3, "EOR", AbsoluteY},
		0x41: {2, "EOR", IndirectX},
		0x51: {2, "EOR", IndirectY},

		0x18: {1, "CLC", None},
		0x38: {1, "SEC", None},
		0x58: {1, "CLI", None},
		0x78: {1, "SEI", None},
		0xB8: {1, "CLV", None},
		0xD8: {1, "CLD", None},
		0xF8: {1, "SED", None},

		0xE6: {2, "INC", ZeroPage},
		0xF6: {2, "INC", ZeroPageX},
		0xEE: {3, "INC", Absolute},
		0xFE: {3, "INC", AbsoluteX},

		0x4C: {3, "JMP", Absolute}, // special cased when printing
		0x6C: {3, "JMP", Indirect},

		0x20: {3, "JSR", Absolute}, // special cased when printing

		0xA9: {2, "LDA", Immediate},
		0xA5: {2, "LDA", ZeroPage},
		0xB5: {2, "LDA", ZeroPageX},
		0xAD: {3, "LDA", Absolute},
		0xBD: {3, "LDA", AbsoluteX},
		0xB9: {3, "LDA", AbsoluteY},
		0xA1: {2, "LDA", IndirectX},
		0xB1: {2, "LDA", IndirectY},

		0xA2: {2, "LDX", Immediate},
		0xA6: {2, "LDX", ZeroPage},
		0xB6: {2, "LDX", ZeroPageY},
		0xAE: {3, "LDX", Absolute},
		0xBE: {3, "LDX", AbsoluteY},

		0xA0: {2, "LDY", Immediate},
		0xA4: {2, "LDY", ZeroPage},
		0xB4: {2, "LDY", ZeroPageX},
		0xAC: {3, "LDY", Absolute},
		0xBC: {3, "LDY", AbsoluteX},

		0x4A: {1, "LSR", Accumulator},
		0x46: {2, "LSR", ZeroPage},
		0x56: {2, "LSR", ZeroPageX},
		0x4E: {3, "LSR", Absolute},
		0x5E: {3, "LSR", AbsoluteX},

		0xEA: {1, "NOP", None},

		0x09: {2, "ORA", Immediate},
		0x05: {2, "ORA", ZeroPage},
		0x15: {2, "ORA", ZeroPageX},
		0x0D: {3, "ORA", Absolute},
		0x1D: {3, "ORA", AbsoluteX},
		0x19: {3, "ORA", AbsoluteY},
		0x01: {2, "ORA", IndirectX},
		0x11: {2, "ORA", IndirectY},

		0xAA: {1, "TAX", None},
		0x8A: {1, "TXA", None},
		0xCA: {1, "DEX", None},
		0xE8: {1, "INX", None},
		0xA8: {1, "TAY", None},
		0x98: {1, "TYA", None},
		0x88: {1, "DEY", None},
		0xC8: {1, "INY", None},

		0x2A: {1, "ROL", Accumulator},
		0x26: {2, "ROL", ZeroPage},
		0x36: {2, "ROL", ZeroPageX},
		0x2E: {3, "ROL", Absolute},
		0x3E: {3, "ROL", AbsoluteX},

		0x6A: {1, "ROR", Accumulator},
		0x66: {2, "ROR", ZeroPage},
		0x76: {2, "ROR", ZeroPageX},
		0x6E: {3, "ROR", Absolute},
		0x7E: {3, "ROR", AbsoluteX},

		0x40: {1, "RTI", None},

		0x60: {1, "RTS", None},

		0xE9: {2, "SBC", Immediate},
		0xE5: {2, "SBC", ZeroPage},
		0xF5: {2, "SBC", ZeroPageX},
		0xED: {3, "SBC", Absolute},
		0xFD: {3, "SBC", AbsoluteX},
		0xF9: {3, "SBC", AbsoluteY},
		0xE1: {2, "SBC", IndirectX},
		0xF1: {2, "SBC", IndirectY},

		0x47: {2, "SRE", ZeroPage},
		0x57: {2, "SRE", ZeroPageX},
		0x4F: {3, "SRE", Absolute},
		0x5F: {3, "SRE", AbsoluteX},
		0x5B: {3, "SRE", AbsoluteY},
		0x43: {2, "SRE", IndirectX},
		0x53: {2, "SRE", IndirectY},

		0x85: {2, "STA", ZeroPage},
		0x95: {2, "STA", ZeroPageX},
		0x8D: {3, "STA", Absolute},
		0x9D: {3, "STA", AbsoluteX},
		0x99: {3, "STA", AbsoluteY},
		0x81: {2, "STA", IndirectX},
		0x91: {2, "STA", IndirectY},

		0x9A: {1, "TXS", None},
		0xBA: {1, "TSX", None},
		0x48: {1, "PHA", None},
		0x68: {1, "PLA", None},
		0x08: {1, "PHP", None},
		0x28: {1, "PLP", None},

		0x07: {2, "SLO", ZeroPage},
		0x17: {2, "SLO", ZeroPageX},
		0x0F: {3, "SLO", Absolute},
		0x1F: {3, "SLO", AbsoluteX},
		0x1B: {3, "SLO", AbsoluteY},
		0x03: {2, "SLO", IndirectX},
		0x13: {2, "SLO", IndirectY},

		0x86: {2, "STX", ZeroPage},
		0x96: {2, "STX", ZeroPageY},
		0x8E: {3, "STX", Absolute},

		0x84: {2, "STY", ZeroPage},
		0x94: {2, "STY", ZeroPageX},
		0x8C: {3, "STY", Absolute},
	}

	// UndocumentedInstructions is not exhaustive and only tracks the opcodes
	// that are included in OpCodesMap.
	UndocumentedInstructions = []string{"ANC", "SRE", "SLO"}

	branchInstructions = []string{"BPL", "BMI", "BVC", "BVS", "BCC", "BCS", "BNE", "BEQ"}

	jumpInstructions = []string{"JMP", "JSR"}

	// Maps absolute addresses to names of BBC MICRO OS calls
	addressToOsCallName = map[uint]string{
		0xFFB9: "OSRDRM",
		0xFFBF: "OSEVEN",
		0xFFC2: "GSINIT",
		0xFFC5: "GSREAD",
		0xFFC8: "NVRDCH", // non-vectored OSRDCH
		0xFFCB: "NVWRCH", // non-vectored OSWRCH
		0xFFCE: "OSFIND",
		0xFFE0: "OSRDCH",
		0xFFE3: "OSASCI",
		0xFFE7: "OSNEWL",
		0xFFEE: "OSWRCH",
		0xFFF1: "OSWORD",
		0xFFF4: "OSBYTE",
		0xFFF7: "OSCLI",
	}

	// Maps OS Vector Addresses to string identifiers
	osVectorAddresses = map[uint]string{
		0x200: "USERV",
		0x202: "BRKV",
		0x204: "IRQ1V",
		0x206: "IRQ2V",
		0x208: "CLIV",
		0x20A: "BYTEV",
		0x20C: "WORDV",
		0x20E: "WRCHV",
		0x210: "RDCHV",
		0x212: "FILEV",
		0x214: "ARGV",
		0x216: "BGETV",
		0x218: "BPUTV",
		0x21A: "GBPBV",
		0x21C: "FINDV",
		0x21E: "FSCV",
		0x220: "EVENTV",
		0x222: "UPTV",
		0x224: "NETV",
		0x226: "VDUV",
		0x228: "KEYV",
		0x22A: "INSV",
		0x22C: "REMV",
		0x22E: "CNPV",
		0x230: "IND1V", // Not documented in BBC Micro AUG
		0x232: "IND2V",
		0x234: "IND3V",
	}

	branchTargets map[uint]int
	usedOSAddress map[uint]bool
	usedOSVector  map[uint]bool
)

type branchType int

const (
	Neither branchType = iota
	Branch
	Jump
)

func (o *opcode) branchOrJump() branchType {
	for _, v := range branchInstructions {
		if o.name == v {
			return Branch
		}
	}

	for _, v := range jumpInstructions {
		if o.name == v {
			return Jump
		}
	}

	return Neither
}

func findBranchTargets(program []uint8, maxBytes, offset, branchAdjust uint) {
	// Track all reachable instructions. That is the address of the first
	// opcode of each instruction starting at offset and moving forwards.
	iloc := make(map[uint]bool)

	branchTargets = make(map[uint]int)
	cursor := offset
	for cursor < (offset + maxBytes) {
		iloc[cursor+branchAdjust] = true // Reachable instruction
		b := program[cursor]

		if op, ok := OpCodesMap[b]; ok {
			instructions := program[cursor : cursor+op.length]

			switch op.branchOrJump() {
			case Branch:
				// This is ugly but it will do for now
				offset := int(instructions[1]) // All branches are 2 bytes long
				if offset > 127 {
					offset = offset - 256
				}
				// Adjust offset to account for the 2 byte behavior, see
				// genBranch().
				offset += 2

				targ := cursor + uint(offset) + branchAdjust
				if _, ok := branchTargets[targ]; !ok {
					branchTargets[targ] = 0 // value will be filled out later
				}
			case Jump:
				// Skip indirect jump since we don't know the target of the jump
				if b != 0x6C {
					targ := (uint(instructions[2]) << 8) + uint(instructions[1])
					if _, ok := branchTargets[targ]; !ok {
						branchTargets[targ] = 0 // value will be filled out later
					}

					// If the jump target is a well known OS call then mark as seen
					if _, ok := addressToOsCallName[targ]; ok {
						usedOSAddress[targ] = true
					}
				}
			case Neither:
				// Check instructions with Absolute addressing
				if op.addrMode == Absolute {
					targ := (uint(instructions[2]) << 8) + uint(instructions[1])
					if _, ok := osVectorAddresses[targ]; ok {
						usedOSVector[targ] = true
					}
				}
			}

			cursor += op.length
		} else {
			cursor++
		}
	}

	// Reject branch targets that point to unreachable instructions. This can
	// happen disassembling data and the byte values generate a branch
	// instruction with a relative address that does not point to the beginning
	// of a reachable instruction.
	for k := range branchTargets {
		if _, ok := iloc[k]; !ok {
			delete(branchTargets, k)
		}
	}

	// Sort branch targets in order of increasing address
	bt := make([]int, len(branchTargets))
	i := 0
	for k := range branchTargets {
		bt[i] = int(k)
		i++
	}
	sort.Ints(bt)
	for i, v := range bt {
		branchTargets[uint(v)] = i
	}
}

func decode(op opcode, bytes []byte, cursor, branchAdjust uint) string {
	// Jump and Branch instructions have special handling
	if bytes[0] == 0x4C || bytes[0] == 0x20 {
		// JMP &1234 and JSR &1234 are special cased with naming for well known
		// OS call entry points.
		return genAbsoluteOsCall(bytes)
	}
	if op.branchOrJump() == Branch {
		return genBranch(bytes, cursor, branchAdjust)
	}

	switch op.addrMode {
	case None:
		return ""
	case Accumulator:
		return "A"
	case Immediate:
		return fmt.Sprintf("#&%02X", bytes[1])
	case Absolute:
		val := (uint(bytes[2]) << 8) + uint(bytes[1])

		// Look up in the OS vector address space
		if osv, ok := osVectorAddresses[val]; ok {
			return osv
		}
		// Try again with the bottom bit cleared because each vector is 16-bit
		// eg. USERV vector is at 0x200 and 0x201.
		if osv, ok := osVectorAddresses[val&^uint(1)]; ok {
			return osv + "+1"
		}

		// Unrecognized address, return as numeric
		return fmt.Sprintf("&%04X", val)
	case ZeroPage:
		return fmt.Sprintf("&%02X", bytes[1])
	case ZeroPageX:
		return fmt.Sprintf("&%02X,X", bytes[1])
	case ZeroPageY:
		return fmt.Sprintf("&%02X,Y", bytes[1])
	case Indirect:
		val := (uint(bytes[2]) << 8) + uint(bytes[1])
		return fmt.Sprintf("(&%04X)", val)
	case AbsoluteX:
		val := (uint(bytes[2]) << 8) + uint(bytes[1])
		return fmt.Sprintf("&%04X,X", val)
	case AbsoluteY:
		val := (uint(bytes[2]) << 8) + uint(bytes[1])
		return fmt.Sprintf("&%04X,Y", val)
	case IndirectX:
		return fmt.Sprintf("(&%02X,X)", bytes[1])
	case IndirectY:
		return fmt.Sprintf("(&%02X),Y", bytes[1])
	default:
		return "UNKNOWN ADDRESS MODE"
	}
}

func genAbsoluteOsCall(bytes []byte) string {
	addr := (uint(bytes[2]) << 8) + uint(bytes[1])

	// Check if it is a well known OS address
	if osCall, ok := addressToOsCallName[addr]; ok {
		return osCall
	}

	// Check if it is a known branch target
	if targetIdx, ok := branchTargets[addr]; ok {
		return fmt.Sprintf(labelFormatString, targetIdx)
	}

	return fmt.Sprintf("&%04X", addr)
}

func genBranch(bytes []byte, cursor, branchAdjust uint) string {
	// From http://www.6502.org/tutorials/6502opcodes.html
	// "When calculating branches a forward branch of 6 skips the following 6
	// bytes so, effectively the program counter points to the address that is 8
	// bytes beyond the address of the branch opcode; and a backward branch of
	// $FA (256-6) goes to an address 4 bytes before the branch instruction."
	offset := int(bytes[1]) // All branches are 2 bytes long
	if offset > 127 {
		offset = offset - 256
	}
	// Adjust offset to account for the 2 byte behavior from the comment block
	// above.
	offset += 2

	targetAddr := cursor + uint(offset) + branchAdjust
	// TODO: Explore branch relative offset in the end of line comment

	targetIdx, ok := branchTargets[targetAddr]
	if !ok {
		// If the branch offset is not a 'reachable' instruction then express
		// the branch with the relative offset. However beebasm interprets an
		// integer literal as an absolute address, so instead write out an
		// expression that generates the same opcodes, e.g. P%+12 or P%-87
		return fmt.Sprintf("P%%%+d", offset)
	}
	return fmt.Sprintf(labelFormatString, targetIdx)
}
