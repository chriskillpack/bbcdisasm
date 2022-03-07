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

		sb.WriteByte(' ')

		// All instructions are at least one byte long and the first byte is
		// sufficient to identify the opcode.
		b := program[cursor]

		// Situations that can arise decoding the next instruction
		// 1) If the byte does not match an opcode - print as data
		// 2) If the byte matches a documented opcode:
		//      If the instruction won't assemble identically then print as data
		//      Otherwise, decode operands and print
		// 3) If the byte matches an undocumented opcode:
		//      Retrieve operands, print as data, mark UD
		op, ok := OpCodesMap[b]
		if ok {
			instruction := program[cursor : cursor+op.Length]
			doc := isOpcodeDocumented(op)
			wai := willAssembleIdentically(op, instruction)
			if doc && wai {
				// If here then documented instruction that will assemble correctly
				printInstruction(&sb, op, instruction, cursor, branchAdjust)

				cursor += op.Length
			} else {
				// The instruction is undocumented or beebasm will not assemble to the same bytes,
				// so the instruction is treated as data.
				printData(&sb, instruction, cursor+branchAdjust)

				if !doc {
					// Undocumented instruction includes additional info before printable bytes
					// EQUB [opcode],...,[opcode] \ [address] UD [instruction mnemonic]   [printable bytes]
					//                            ^--- 25th column                        ^--- 45th column
					sb.WriteString("UD ")
					sb.WriteString(op.Name)
				}

				appendPrintableBytes(&sb, instruction)

				cursor += uint(len(instruction))
			}
		} else {
			bs := []byte{b}
			printData(&sb, bs, cursor+branchAdjust)
			appendPrintableBytes(&sb, bs)
			cursor++
		}

		sb.WriteByte('\n')
		w.Write([]byte(sb.String()))
	}
}

func printInstruction(sb *strings.Builder, op Opcode, instruction []byte, cursor, branchAdjust uint) {
	// A valid instruction will be printed to a line with format
	//
	// [instruction mnemonic]     \ [address] [instruction opcodes]   [printable bytes]
	//                            ^--- 25th column                    ^--- 45th column
	sb.WriteString(op.Name)
	sb.WriteByte(' ')
	sb.WriteString(decode(op, instruction, cursor, branchAdjust))

	appendSpaces(sb, max(24-sb.Len(), 1))
	sb.WriteString("\\ ")

	out := []string{
		fmt.Sprintf("&%04X", cursor+branchAdjust),
	}
	for _, i := range instruction {
		out = append(out, fmt.Sprintf("%02X", i))
	}
	sb.WriteString(strings.Join(out, " "))

	appendPrintableBytes(sb, instruction)
}

// Print data in hex as comma-delimited EQUB statement. Assumes that there are
// between 1 and 3 data bytes though it will handle any amount.
func printData(sb *strings.Builder, data []byte, address uint) {
	// Data will be printed to a line with format
	// EQUB &[byte],...,&[byte]    \ [address] [opcode]   [printable bytes]
	//                             ^--- 25th column       ^--- 45th column
	var out []string
	for _, i := range data {
		out = append(out, fmt.Sprintf("&%02X", i))
	}
	sb.WriteString("EQUB ")
	sb.WriteString(strings.Join(out, ","))

	appendSpaces(sb, max(24-sb.Len(), 1))
	sb.WriteString("\\ ")
	sb.WriteString(fmt.Sprintf("&%04X", address))
	sb.WriteByte(' ')
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

// willAssembleIdentically checks if beebasm will assemble the instruction as written
//
// Given an instruction with a 16-bit absolute address operand that lies in the
// Zero Page e.g. LDA &0012, beebasm will instead assemble using the zero page
// form if supported, e.g. LDA &12. This behavior breaks binary compatibility.
func willAssembleIdentically(op Opcode, instruction []byte) bool {
	if op.AddrMode == Absolute || op.AddrMode == AbsoluteX || op.AddrMode == AbsoluteY {
		tgt := (uint(instruction[2]) << 8) + uint(instruction[1])
		if tgt < 0x100 {
			return false
		}
	}

	return true
}

var disasmHeader = `\ ******************************************************************************
\
\ This disassembly was produced by bbcdisasm
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
