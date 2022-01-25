package bbcdisasm

import (
	"fmt"
	"sort"
)

type decodeFunc func(bytes []byte, cursor uint) string

type opcode struct {
	length uint   // number of bytes include opcode
	name   string // human readable instruction 'name' of opcode
	decode decodeFunc
}

var branchOffset uint // Adjustment to make to absolute branch locations

var (
	// OpCodesMap maps from first instruction opcode byte to 6502 instruction
	// Most opcodes from http://www.6502.org/tutorials/6502opcodes.html
	// ANC, SLO, SRE from https://github.com/mattgodbolt/jsbeeb/blob/master/6502.opcodes.js
	OpCodesMap = map[byte]opcode{
		0x69: {2, "ADC", genImmediate},
		0x65: {2, "ADC", genZeroPage},
		0x75: {2, "ADC", genZeroPageX},
		0x6D: {3, "ADC", genAbsolute},
		0x7D: {3, "ADC", genAbsoluteX},
		0x79: {3, "ADC", genAbsoluteY},
		0x61: {2, "ADC", genIndirectX},
		0x71: {2, "ADC", genIndirectY},

		0x0B: {2, "ANC", genImmediate},
		0x2B: {2, "ANC", genImmediate},

		0x29: {2, "AND", genImmediate},
		0x25: {2, "AND", genZeroPage},
		0x35: {2, "AND", genZeroPageX},
		0x2D: {3, "AND", genAbsolute},
		0x3D: {3, "AND", genAbsoluteX},
		0x39: {3, "AND", genAbsoluteY},
		0x21: {2, "AND", genIndirectX},
		0x31: {2, "AND", genIndirectY},

		0x0A: {1, "ASL", genAccumulator},
		0x06: {2, "ASL", genZeroPage},
		0x16: {2, "ASL", genZeroPageX},
		0x0E: {3, "ASL", genAbsolute},
		0x1E: {3, "ASL", genAbsoluteX},

		0x24: {2, "BIT", genZeroPage},
		0x2C: {3, "BIT", genAbsolute},

		0x10: {2, "BPL", genBranch},
		0x30: {2, "BMI", genBranch},
		0x50: {2, "BVC", genBranch},
		0x70: {2, "BVS", genBranch},
		0x90: {2, "BCC", genBranch},
		0xB0: {2, "BCS", genBranch},
		0xD0: {2, "BNE", genBranch},
		0xF0: {2, "BEQ", genBranch},

		0x00: {1, "BRK", genNull},

		0xC9: {2, "CMP", genImmediate},
		0xC5: {2, "CMP", genZeroPage},
		0xD5: {2, "CMP", genZeroPageX},
		0xCD: {3, "CMP", genAbsolute},
		0xDD: {3, "CMP", genAbsoluteX},
		0xD9: {3, "CMP", genAbsoluteY},
		0xC1: {2, "CMP", genIndirectX},
		0xD1: {2, "CMP", genIndirectY},

		0xE0: {2, "CPX", genImmediate},
		0xE4: {2, "CPX", genZeroPage},
		0xEC: {3, "CPX", genAbsolute},

		0xC0: {2, "CPY", genImmediate},
		0xC4: {2, "CPY", genZeroPage},
		0xCC: {3, "CPY", genAbsolute},

		0xC6: {2, "DEC", genZeroPage},
		0xD6: {2, "DEC", genZeroPageX},
		0xCE: {3, "DEC", genAbsolute},
		0xDE: {3, "DEC", genAbsoluteX},

		0x49: {2, "EOR", genImmediate},
		0x45: {2, "EOR", genZeroPage},
		0x55: {2, "EOR", genZeroPageX},
		0x4D: {3, "EOR", genAbsolute},
		0x5D: {3, "EOR", genAbsoluteX},
		0x59: {3, "EOR", genAbsoluteY},
		0x41: {2, "EOR", genIndirectX},
		0x51: {2, "EOR", genIndirectY},

		0x18: {1, "CLC", genNull},
		0x38: {1, "SEC", genNull},
		0x58: {1, "CLI", genNull},
		0x78: {1, "SEI", genNull},
		0xB8: {1, "CLV", genNull},
		0xD8: {1, "CLD", genNull},
		0xF8: {1, "SED", genNull},

		0xE6: {2, "INC", genZeroPage},
		0xF6: {2, "INC", genZeroPageX},
		0xEE: {3, "INC", genAbsolute},
		0xFE: {3, "INC", genAbsoluteX},

		0x4C: {3, "JMP", genAbsoluteOsCall},
		0x6C: {3, "JMP", genIndirect},

		0x20: {3, "JSR", genAbsoluteOsCall},

		0xA9: {2, "LDA", genImmediate},
		0xA5: {2, "LDA", genZeroPage},
		0xB5: {2, "LDA", genZeroPageX},
		0xAD: {3, "LDA", genAbsolute},
		0xBD: {3, "LDA", genAbsoluteX},
		0xB9: {3, "LDA", genAbsoluteY},
		0xA1: {2, "LDA", genIndirectX},
		0xB1: {2, "LDA", genIndirectY},

		0xA2: {2, "LDX", genImmediate},
		0xA6: {2, "LDX", genZeroPage},
		0xB6: {2, "LDX", genZeroPageY},
		0xAE: {3, "LDX", genAbsolute},
		0xBE: {3, "LDX", genAbsoluteY},

		0xA0: {2, "LDY", genImmediate},
		0xA4: {2, "LDY", genZeroPage},
		0xB4: {2, "LDY", genZeroPageX},
		0xAC: {3, "LDY", genAbsolute},
		0xBC: {3, "LDY", genAbsoluteX},

		0x4A: {1, "LSR", genAccumulator},
		0x46: {2, "LSR", genZeroPage},
		0x56: {2, "LSR", genZeroPageX},
		0x4E: {3, "LSR", genAbsolute},
		0x5E: {3, "LSR", genAbsoluteX},

		0xEA: {1, "NOP", genNull},

		0x09: {2, "ORA", genImmediate},
		0x05: {2, "ORA", genZeroPage},
		0x15: {2, "ORA", genZeroPageX},
		0x0D: {3, "ORA", genAbsolute},
		0x1D: {3, "ORA", genAbsoluteX},
		0x19: {3, "ORA", genAbsoluteY},
		0x01: {2, "ORA", genIndirectX},
		0x11: {2, "ORA", genIndirectY},

		0xAA: {1, "TAX", genNull},
		0x8A: {1, "TXA", genNull},
		0xCA: {1, "DEX", genNull},
		0xE8: {1, "INX", genNull},
		0xA8: {1, "TAY", genNull},
		0x98: {1, "TYA", genNull},
		0x88: {1, "DEY", genNull},
		0xC8: {1, "INY", genNull},

		0x2A: {1, "ROL", genAccumulator},
		0x26: {2, "ROL", genZeroPage},
		0x36: {2, "ROL", genZeroPageX},
		0x2E: {3, "ROL", genAbsolute},
		0x3E: {3, "ROL", genAbsoluteX},

		0x6A: {1, "ROR", genAccumulator},
		0x66: {2, "ROR", genZeroPage},
		0x76: {2, "ROR", genZeroPageX},
		0x6E: {3, "ROR", genAbsolute},
		0x7E: {3, "ROR", genAbsoluteX},

		0x40: {1, "RTI", genNull},

		0x60: {1, "RTS", genNull},

		0xE9: {2, "SBC", genImmediate},
		0xE5: {2, "SBC", genZeroPage},
		0xF5: {2, "SBC", genZeroPageX},
		0xED: {3, "SBC", genAbsolute},
		0xFD: {3, "SBC", genAbsoluteX},
		0xF9: {3, "SBC", genAbsoluteY},
		0xE1: {2, "SBC", genIndirectX},
		0xF1: {2, "SBC", genIndirectY},

		0x47: {2, "SRE", genZeroPage},
		0x57: {2, "SRE", genZeroPageX},
		0x4F: {3, "SRE", genAbsolute},
		0x5F: {3, "SRE", genAbsoluteX},
		0x5B: {3, "SRE", genAbsoluteY},
		0x43: {2, "SRE", genIndirectX},
		0x53: {2, "SRE", genIndirectY},

		0x85: {2, "STA", genZeroPage},
		0x95: {2, "STA", genZeroPageX},
		0x8D: {3, "STA", genAbsolute},
		0x9D: {3, "STA", genAbsoluteX},
		0x99: {3, "STA", genAbsoluteY},
		0x81: {2, "STA", genIndirectX},
		0x91: {2, "STA", genIndirectY},

		0x9A: {1, "TXS", genNull},
		0xBA: {1, "TSX", genNull},
		0x48: {1, "PHA", genNull},
		0x68: {1, "PLA", genNull},
		0x08: {1, "PHP", genNull},
		0x28: {1, "PLP", genNull},

		0x07: {2, "SLO", genZeroPage},
		0x17: {2, "SLO", genZeroPageX},
		0x0F: {3, "SLO", genAbsolute},
		0x1F: {3, "SLO", genAbsoluteX},
		0x1B: {3, "SLO", genAbsoluteY},
		0x03: {2, "SLO", genIndirectX},
		0x13: {2, "SLO", genIndirectY},

		0x86: {2, "STX", genZeroPage},
		0x96: {2, "STX", genZeroPageY},
		0x8E: {3, "STX", genAbsolute},

		0x84: {2, "STY", genZeroPage},
		0x94: {2, "STY", genZeroPageY},
		0x8C: {3, "STY", genAbsolute},
	}

	// Record which instructions are undocumented
	// This list is not exhaustive and only tracks the undocumented opcodes
	// that are included in OpCodesMap.
	UndocumentedInstructions = []string{"ANC", "SRE", "SLO"}

	branchInstructions = []string{"BPL", "BMI", "BVC", "BVS", "BCC", "BCS", "BNE", "BEQ"}

	// Maps absolute addresses to names of BBC MICRO OS calls
	addressToOsCallName = map[uint]string{
		0xFFB9: "OSRDRM",
		0xFFBF: "OSEVEN",
		0xFFC2: "GSINIT",
		0xFFC5: "GSREAD",
		0xFFEE: "OSWRCH",
		0xFFE0: "OSRDCH",
		0xFFE7: "OSNEWL",
		0xFFE3: "OSASCI",
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
		// 0x230 is not documented in BBC Micro AUG
		0x232: "IND2V",
		0x234: "IND3V",
	}

	branchTargets = []int{}
)

// Returns true if this opcode is a branch
func (o *opcode) isBranch() bool {
	for _, v := range branchInstructions {
		if o.name == v {
			return true
		}
	}

	return false
}

// If addr matches a branch target return it's index in the
// targets array, -1 if no match
func branchTargetForAddr(addr uint) int {
	for i, bt := range branchTargets {
		if addr == uint(bt) {
			return i
		}
	}

	return -1
}

func findBranchTargets(program []uint8, maxBytes, offset uint) {
	branchTargets = []int{}

	cursor := offset
	for cursor < (offset + maxBytes) {
		b := program[cursor]

		if op, ok := OpCodesMap[b]; ok {
			if op.isBranch() {
				// This is ugly but it will do for now
				instructions := program[cursor : cursor+op.length]

				offset := int(instructions[1]) + 2 // All branches are 2 bytes long
				if offset > 127 {
					offset = offset - 256
				}
				branchTargets = append(branchTargets, int(cursor+uint(offset)))
			}
			cursor += op.length
		} else {
			cursor++
		}
	}

	sort.Ints(branchTargets)
}

func genImmediate(bytes []byte, _ uint) string {
	return fmt.Sprintf("#&%02X", bytes[1])
}

func genZeroPage(bytes []byte, _ uint) string {
	return fmt.Sprintf("&%02X", bytes[1])
}

func genZeroPageX(bytes []byte, _ uint) string {
	return fmt.Sprintf("&%02X,X", bytes[1])
}

func genZeroPageY(bytes []byte, _ uint) string {
	return fmt.Sprintf("&%02X,Y", bytes[1])
}

func genAbsolute(bytes []byte, _ uint) string {
	val := (uint(bytes[2]) << 8) + uint(bytes[1])
	return fmt.Sprintf("&%04X", val)
}

func genAbsoluteOsCall(bytes []byte, _ uint) string {
	val := (uint(bytes[2]) << 8) + uint(bytes[1])
	if osCall, ok := addressToOsCallName[val]; ok {
		return osCall
	} else {
		return fmt.Sprintf("&%04X", val)
	}
}

func genAbsoluteX(bytes []byte, _ uint) string {
	val := (uint(bytes[2]) << 8) + uint(bytes[1])
	return fmt.Sprintf("&%04X,X", val)
}

func genAbsoluteY(bytes []byte, _ uint) string {
	val := (uint(bytes[2]) << 8) + uint(bytes[1])
	return fmt.Sprintf("&%04X,Y", val)
}

func genIndirect(bytes []byte, _ uint) string {
	val := (uint(bytes[2]) << 8) + uint(bytes[1])
	return fmt.Sprintf("(&%04X)", val)
}

func genIndirectX(bytes []byte, _ uint) string {
	return fmt.Sprintf("(&%02X,X)", bytes[1])
}

func genIndirectY(bytes []byte, _ uint) string {
	return fmt.Sprintf("(&%02X),Y", bytes[1])
}

func genBranch(bytes []byte, cursor uint) string {
	// From http://www.6502.org/tutorials/6502opcodes.html
	// "When calculating branches a forward branch of 6 skips the following 6
	// bytes so, effectively the program counter points to the address that is 8
	// bytes beyond the address of the branch opcode; and a backward branch of $FA
	// (256-6) goes to an address 4 bytes before the branch instruction."
	offset := int(bytes[1]) + 2 // All branches are 2 bytes long
	if offset > 127 {
		offset = offset - 256
	}
	targetAddr := cursor + uint(offset)
	// TODO: Explore branch relative offset in the end of line comment

	targetIdx := branchTargetForAddr(targetAddr)
	if targetIdx == -1 {
		panic("Target address was not found in first pass")
	}
	return fmt.Sprintf("loop_%d", targetIdx)
}

func genAccumulator([]byte, uint) string {
	return "A"
}

func genNull([]byte, uint) string {
	return ""
}
