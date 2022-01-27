package bbcdisasm

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"
)

// DiskImage represents an Acorn DFS disk image
type DiskImage struct {
	Title   string
	Sectors int
	BootOpt int
	Cycle   int
	Files   []Catalog
}

// Catalog represents a file in Acorn DFS
type Catalog struct {
	Filename    string
	Dir         string
	Length      int
	LoadAddr    int
	ExecAddr    int
	StartSector int
	Attr        byte
}

// Disassemble prints a 6502 program to stdout
// loadAddr fixes up addresses to match the load address. Uses
// https://twitter.com/KevEdwardsRetro/status/996474534730567681 as an output template
func Disassemble(program []byte, maxBytes, offset, loadAddr uint, w io.Writer) {
	// First pass through program is to find the location
	// of any branches. These will be marked as labels in
	// the output.
	findBranchTargets(program, maxBytes, offset)

	distem, _ := template.New("disasm").Parse(disasmHeader)
	data := struct {
		OSmap    map[uint]string
		LoadAddr uint
	}{addressToOsCallName, loadAddr}
	distem.Execute(w, data)

	branchOffset = loadAddr // gross. setting a package level var

	// Second pass through program is to decode each instruction
	// and print to stdout.
	cursor := offset
	for cursor < (offset + maxBytes) {
		if targetIdx, ok := branchTargets[cursor]; ok {
			fmt.Fprintf(w, ".loop_%d\n", targetIdx)
		}

		// All instructions are at least one byte long and the first
		// byte is sufficient to identify the instruction
		b := program[cursor]

		var sb strings.Builder
		sb.WriteByte(' ')

		op, ok := OpCodesMap[b]
		if ok && isOpcodeDocumented(op) {
			// The valid instruction will be printed to a line with format
			//
			// address [instruction op codes ...] decoded instruction
			opcodes := program[cursor : cursor+op.length]

			sb.WriteString(op.name)
			sb.WriteByte(' ')
			sb.WriteString(op.decode(opcodes, cursor))
			if sb.Len() < 25 {
				// 25 -1 (for leading space below) -1 (so that \ is at col 25)
				sb.Write(bytes.Repeat([]byte{' '}, 23-sb.Len()))
			}
			sb.WriteString(" \\ ")

			out := []string{
				fmt.Sprintf("&%04X", cursor+uint(branchOffset)),
			}
			for _, i := range opcodes {
				out = append(out, fmt.Sprintf("%02X", i))
			}
			sb.WriteString(strings.Join(out, " "))

			cursor += op.length
		} else {
			bs := []byte{b}
			if ok {
				bs = program[cursor : cursor+op.length]
			}

			var out []string
			for _, i := range bs {
				out = append(out, fmt.Sprintf("&%02X", i))
			}
			sb.WriteString("EQUB ")
			sb.WriteString(strings.Join(out, ","))

			if ok {
				if sb.Len() < 25 {
					sb.Write(bytes.Repeat([]byte{' '}, 23-sb.Len()))
				}
				sb.WriteString(" \\ UD ")
				sb.WriteString(op.name)
			}
			cursor += uint(len(bs))
		}
		sb.WriteByte('\n')
		w.Write([]byte(sb.String()))
	}
}

// ParseDFS reads the disk and file catalogs from binary data
// Resources
//   http://mdfs.net/Docs/Comp/Disk/Format/DFS
//   http://chrisacorns.computinghistory.org.uk/docs/Acorn/Manuals/Acorn_DiscSystemUGI2.pdf
func ParseDFS(dfs []byte) *DiskImage {
	img := &DiskImage{}

	nfiles := int(dfs[0x105]) / 8
	img.Title = strings.TrimRight(string(dfs[0:8])+string(dfs[0x100:0x104]), "\000")
	img.Sectors = int(dfs[0x107]) + int(dfs[0x106]&3)*256
	img.BootOpt = int(dfs[0x106]&48) >> 4
	img.Cycle = int(dfs[0x104])
	img.Files = make([]Catalog, nfiles)

	// Read file catalog entries
	for i := 0; i < nfiles; i++ {
		file := &img.Files[i]

		// Read out the filename
		var offset int
		offset = 0x008 + i*8
		file.Filename, file.Attr = readFilename(dfs[offset : offset+7])
		file.Dir = string(dfs[offset+7])

		// Read file info
		offset = 0x108 + i*8
		file.Length = int(dfs[offset+4]) + int(dfs[offset+5])*256 + int(dfs[offset+6]&0b110000)*4096
		file.LoadAddr = int(dfs[offset+0]) + int(dfs[offset+1])*256 + int(dfs[offset+6]&0b1100)*16384
		file.ExecAddr = int(dfs[offset+2]) + int(dfs[offset+3])*256 + int(dfs[offset+6]&0b11000000)*1024
		file.StartSector = int(dfs[offset+7]) + int(dfs[offset+6]&0b11)*256
	}

	return img
}

func readFilename(block []byte) (string, byte) {
	if len(block) < 7 {
		panic("block is too short")
	}

	// Read out file attributes, stored in the top bit of filename characters,
	// and clear out for a printable ASCII filename.
	name := make([]byte, len(block))
	var attr byte
	for i, v := range block {
		attr |= (v & 0x80) >> (7 - i)
		name[i] = v & 0x7f
	}

	return strings.TrimRight(string(name), " "), attr
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
