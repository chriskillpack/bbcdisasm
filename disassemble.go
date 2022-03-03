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
	usedOSAddress = make(map[uint]bool)
	usedOSVector = make(map[uint]bool)

	// First pass through program is to find the location
	// of any branches. These will be marked as labels in
	// the output.
	findBranchTargets(program, maxBytes, offset, branchAdjust)

	distem, _ := template.New("disasm").Parse(disasmHeader)
	data := struct {
		UsedOSAddress map[uint]bool
		OSAddress     map[uint]string
		UsedOSVector  map[uint]bool
		OSVector      map[uint]string
		LoadAddr      uint
	}{usedOSAddress, addressToOsCallName, usedOSVector, osVectorAddresses, branchAdjust}
	if err := distem.Execute(w, data); err != nil {
		panic(err)
	}

	// Second pass through program is to decode each instruction
	// and print to stdout.
	cursor := offset
	for cursor < (offset + maxBytes) {
		var sb strings.Builder
		if targetIdx, ok := branchTargets[cursor+branchAdjust]; ok {
			sb.WriteByte('.')
			sb.WriteString(fmt.Sprintf(labelFormatString, targetIdx))
			sb.WriteString("\n")
			w.Write([]byte(sb.String()))

			sb.Reset()
		}

		// All instructions are at least one byte long and the first
		// byte is sufficient to identify the instruction.
		b := program[cursor]

		sb.WriteByte(' ')

		op, ok := OpCodesMap[b]
		if ok && isOpcodeDocumented(op) {
			// A valid instruction will be printed to a line with format
			//
			// [instruction mnemonic]     \ [address] [instruction opcodes]   [printable bytes]
			//                            ^--- 25th column                    ^--- 45th column
			opcodes := program[cursor : cursor+op.Length]

			sb.WriteString(op.Name)
			sb.WriteByte(' ')
			sb.WriteString(decode(op, opcodes, cursor, branchAdjust))

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

			cursor += op.Length
		} else {
			ud := ok

			// If the opcode is unrecognized then it is treated as data and
			// formatted
			//
			// EQUB &[opcode]    \ [address] [opcode]   [printable bytes]
			//                   ^--- 25th column       ^--- 45th column
			bs := []byte{b}
			if ud {
				// If the opcode is recognized then it must be an undocumented
				// instruction (UD). Formatting
				//
				// EQUB [opcode],...,[opcode] \ [address] UD [instruction mnemonic]   [printable bytes]
				//                            ^--- 25th column                        ^--- 45th column
				bs = program[cursor : cursor+op.Length]
			}

			var out []string
			for _, i := range bs {
				out = append(out, fmt.Sprintf("&%02X", i))
			}
			sb.WriteString("EQUB ")
			sb.WriteString(strings.Join(out, ","))

			appendSpaces(&sb, max(24-sb.Len(), 1))
			sb.WriteString("\\ ")
			sb.WriteString(fmt.Sprintf("&%04X", cursor+branchAdjust))
			sb.WriteByte(' ')

			if ud {
				// Undocumented instruction
				sb.WriteString("UD ")
				sb.WriteString(op.Name)
			} else {
				// Data byte. Print out the data byte for visual consistency
				sb.WriteString(fmt.Sprintf("%02X", bs[0]))
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

func isOpcodeDocumented(op Opcode) bool {
	for _, u := range UndocumentedInstructions {
		if op.Name == u {
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

{{ if .UsedOSAddress }}\ OS Call Addresses
{{ $os := .OSAddress }}
{{- range $addr, $elem := .UsedOSAddress }}{{ printf "%-6s" (index $os $addr) }} = {{ printf "&%0X" $addr }}
{{ end }}
{{- end }}
{{ if .UsedOSVector }}\ OS Vector Addresses
{{ $vec := .OSVector }}
{{- range $addr, $elem := .UsedOSVector }}{{ printf "%-5s" (index $vec $addr) }} = {{ printf "&%0X" $addr }}
{{ end }}
{{- end }}
{{ if .LoadAddr }}CODE% = {{ printf "&%X" .LoadAddr }}

ORG CODE%
{{ else -}}
{{ end }}
`
