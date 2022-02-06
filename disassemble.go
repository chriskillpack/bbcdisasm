package bbcdisasm

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"
)

// Disassemble prints a 6502 program to stdout
// offset is where disassembly starts from the beginning of program.
// branchAdjust is used to adjust the target address of relative branches to a
// 'meaningful' address, typically the load address of the program.
func Disassemble(program []byte, maxBytes, offset, branchAdjust uint, w io.Writer) {
	// First pass through program is to find the location
	// of any branches. These will be marked as labels in
	// the output.
	findBranchTargets(program, maxBytes, offset)

	distem, _ := template.New("disasm").Parse(disasmHeader)
	data := struct {
		OSmap    map[uint]string
		LoadAddr uint
	}{addressToOsCallName, branchAdjust}
	distem.Execute(w, data)

	// Second pass through program is to decode each instruction
	// and print to stdout.
	cursor := offset
	for cursor < (offset + maxBytes) {
		if targetIdx, ok := branchTargets[cursor]; ok {
			fmt.Fprintf(w, ".loop_%d\n", targetIdx)
		}

		// All instructions are at least one byte long and the first
		// byte is sufficient to identify the instruction.
		b := program[cursor]

		var sb strings.Builder
		sb.WriteByte(' ')

		op, ok := OpCodesMap[b]
		if ok && isOpcodeDocumented(op) {
			// A valid instruction will be printed to a line with format
			//
			// [instruction mnemonic]     \ [instruction hex address] [instruction opcodes]   [printable bytes]
			//                            ^--- 25th column                                    ^--- 45th column
			opcodes := program[cursor : cursor+op.length]

			sb.WriteString(op.name)
			sb.WriteByte(' ')
			sb.WriteString(op.decode(opcodes, cursor))

			appendSpaces(&sb, max(24-sb.Len(), 1))
			sb.WriteString("\\ ")

			out := []string{
				fmt.Sprintf("&%04X", cursor+branchAdjust),
			}
			for _, i := range opcodes {
				out = append(out, fmt.Sprintf("%02X", i))
			}
			sb.WriteString(strings.Join(out, " "))

			appendPrintableBytes(&sb, opcodes)

			cursor += op.length
		} else {
			// If the opcode is unrecognized then it is treated as data and
			// formatted
			//
			// EQUB &[opcode in hex]    \                 [printable bytes]
			//                          ^--- 25th column  ^--- 45th column
			bs := []byte{b}
			if ok {
				// If the opcode is recognized then it must be an undocumented
				// instruction (UD). Formatting
				//
				// EQUB [opcode],...,[opcode] \ UD [instruction mnemonic]   [printable bytes]
				//                            ^--- 25th column              ^--- 45th column
				bs = program[cursor : cursor+op.length]
			}

			var out []string
			for _, i := range bs {
				out = append(out, fmt.Sprintf("&%02X", i))
			}
			sb.WriteString("EQUB ")
			sb.WriteString(strings.Join(out, ","))

			appendSpaces(&sb, max(24-sb.Len(), 1))
			sb.WriteString("\\ ")

			if ok {
				// Undocumented instruction
				sb.WriteString("UD ")
				sb.WriteString(op.name)
			}

			appendPrintableBytes(&sb, bs)

			cursor += uint(len(bs))
		}
		sb.WriteByte('\n')
		w.Write([]byte(sb.String()))
	}
}

func appendSpaces(sb *strings.Builder, ns int) {
	sb.Write(bytes.Repeat([]byte{' '}, ns))
}

func appendPrintableBytes(sb *strings.Builder, b []byte) {
	appendSpaces(sb, max(44-sb.Len(), 1))
	for _, c := range b {
		sb.WriteByte(toChar(c))
	}
}

func toChar(b byte) byte {
	if b < 32 || b > 126 {
		return '.'
	}
	return b
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func isOpcodeDocumented(op opcode) bool {
	for _, u := range UndocumentedInstructions {
		if op.name == u {
			return false
		}
	}

	return true
}

var disasmHeader = `\ ******************************************************************************
\
\ This disassembly was produced by bbc-disasm
\
\ ******************************************************************************

{{ range $addr, $os := .OSmap }}
  {{- printf "%-6s" $os }} = {{ printf "&%0X" $addr }}
{{ end }}
{{ if .LoadAddr }}CODE% = {{ printf "&%X" .LoadAddr }}

ORG CODE%
{{ end }}

`
