package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/urfave/cli"
)

// Acorn DFS disk image
type diskImage struct {
	title    string
	nSectors int
	bootOpt  int
	cycle    int
	files    []catalog
}

// Acorn DFS catalog item
type catalog struct {
	filename    string
	dir         string
	length      int
	loadAddr    int
	execAddr    int
	startSector int
	attr        byte
}

var (
	loadAddress = 0
)

// Using https://twitter.com/KevEdwardsRetro/status/996474534730567681 as an output template
func disassemble(program []byte, maxBytes, offset uint) {
	// First pass through program is to find the location
	// of any branches. These will be marked as labels in
	// the output.
	findBranchTargets(program, maxBytes, offset)

	// Second pass through program is to decode each instruction
	// and print to stdout.
	cursor := offset
	for cursor < (offset + maxBytes) {
		targetIdx := branchTargetForAddr(cursor)
		if targetIdx != -1 {
			fmt.Printf("loop_%d:\n", targetIdx)
		}

		// All instructions are at least one byte long and the first
		// byte is sufficient to identify the instruction
		b := program[cursor]

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("$%04X ", cursor+uint(loadAddress)))
		if op, ok := OpCodesMap[b]; ok {
			// The valid instruction will be printed to a line with format
			//
			// address [instruction op codes ...] decoded instruction
			opcodes := program[cursor : cursor+op.length]

			var out []string
			for _, i := range opcodes {
				out = append(out, fmt.Sprintf("%02X", i))
			}
			sb.WriteString(strings.Join(out, " "))
			sb.WriteString("\t")
			sb.WriteString(op.name)

			sb.WriteString(fmt.Sprintf(" %s", op.decode(opcodes, cursor)))
			cursor += op.length
		} else {
			// Gracefully handle unrecognized opcodes
			sb.WriteString(fmt.Sprintf("$%02X", b))
			cursor++
		}
		fmt.Println(sb.String())
	}
}

func listDfs(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading %s\n", file)
		return err
	}

	img := parseDfs(data)
	listImage(img)
	return nil
}

// Resources
//   http://mdfs.net/Docs/Comp/Disk/Format/DFS
//   http://chrisacorns.computinghistory.org.uk/docs/Acorn/Manuals/Acorn_DiscSystemUGI2.pdf
func parseDfs(dfs []byte) diskImage {
	image := diskImage{}

	image.title = strings.TrimRight(string(dfs[0:8])+string(dfs[0x100:0x104]), "")

	nFiles := int(dfs[0x105]) / 8

	image.nSectors = int(dfs[0x107]) + int(dfs[0x106]&3)*256
	image.bootOpt = int(dfs[0x106]&48) >> 4
	image.cycle = int(dfs[0x104])
	image.files = make([]catalog, nFiles)

	// Read file catalog entries
	for i := 0; i < nFiles; i++ {
		file := &image.files[i]

		// Read out the filename
		var offset int
		offset = 0x008 + i*8
		file.filename, file.attr = readFilename(dfs[offset : offset+7])
		file.dir = string(dfs[offset+7])

		// Read file info
		offset = 0x108 + i*8
		file.length = int(dfs[offset+4]) + int(dfs[offset+5])*256 + int(dfs[offset+6]&0b110000)*4096
		file.loadAddr = int(dfs[offset+0]) + int(dfs[offset+1])*256 + int(dfs[offset+6]&0b1100)*16384
		file.execAddr = int(dfs[offset+2]) + int(dfs[offset+3])*256 + int(dfs[offset+6]&0b11000000)*1024
		file.startSector = int(dfs[offset+7]) + int(dfs[offset+6]&0b11)*256
	}

	return image
}

func readFilename(block []byte) (string, byte) {
	if len(block) < 7 {
		panic("block is too short")
	}

	name := make([]byte, len(block))
	var attr byte
	for i, v := range block {
		attr |= (v & 0x80) >> (7 - i)
		name[i] = v & 0x7f
	}

	return strings.TrimRight(string(name), " "), attr
}

func listImage(image diskImage) {
	fmt.Printf("Disk Title  %s\n", image.title)
	fmt.Printf("Num Files   %d\n", len(image.files))
	fmt.Printf("Num Sectors %d\n", image.nSectors)
	fmt.Printf("Boot Option %d\n", image.bootOpt)
	fmt.Printf("Disk Cycle  0x%0X\n\n", image.cycle)

	fmt.Println("Filename  Length LoadAddr ExecAddr Sector")
	for _, file := range image.files {
		fmt.Printf("%-7s   %04X   %08X %08X %3d\n", file.filename, file.length, file.loadAddr, file.execAddr, file.startSector)
	}
}

func extractFromDfs(file, entry, outDir string) error {
	var data []byte
	var err error

	if data, err = ioutil.ReadFile(file); err != nil {
		fmt.Printf("Error reading %s\n", file)
		return err
	}

	img := parseDfs(data)

	// Ensure output directory exists
	if outDir != "" {
		fi, err := os.Stat(outDir)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.Mkdir(outDir, os.ModePerm)
				if err != nil {
					return fmt.Errorf("could not create directory %s: %q", outDir, err)
				}
			} else {
				return err
			}
		} else {
			if !fi.IsDir() {
				return fmt.Errorf("output path %s is not a directory", outDir)
			}
		}
	}

	for _, f := range img.files {
		if f.filename == entry || entry == "" {
			// Retrieve data contents
			offset := f.startSector * 256
			d := data[offset:(offset + f.length)]

			ofn := path.Join(outDir, f.filename)
			if err := ioutil.WriteFile(ofn, d, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func disasmFile(file string, offset, length int64) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading %s", file)
		return err
	}

	disassemble(data, uint(length), uint(offset))
	return nil
}

func fileLength(filename string) (int64, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}

	return fi.Size(), nil
}

func main() {
	app := cli.NewApp()
	app.Name = "bbc-disasm"
	app.Usage = "Tool to extract and disassemble programs from BBC Micro DFS disk images"
	app.Action = func(c *cli.Context) error {
		cli.ShowAppHelp(c)
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:      "list",
			Aliases:   []string{"ls"},
			Usage:     "List a DFS disk image",
			ArgsUsage: "image",
			Action: func(c *cli.Context) error {
				args := c.Args()
				if len(args) < 1 {
					return cli.NewExitError("Insufficient arguments", 1)
				}
				return listDfs(c.Args().First())
			},
		},
		{
			Name:      "extract",
			Aliases:   []string{"x"},
			Usage:     "Extract file from DFS disk image",
			ArgsUsage: "image [entry] [outDir]",
			Action: func(c *cli.Context) error {
				args := c.Args()
				if len(args) < 1 {
					return cli.NewExitError("Insufficient arguments", 1)
				}
				var entry, outDir string
				if len(args) >= 2 {
					entry = args[1]
				}
				if len(args) >= 3 {
					outDir = args[2]
				}
				if err := extractFromDfs(args[0], entry, outDir); err != nil {
					return cli.NewExitError("Could not extract file from image", 1)
				}
				return nil
			},
		},
		{
			Name:      "disasm",
			Aliases:   []string{"d"},
			Usage:     "Disassemble a file",
			ArgsUsage: "file [offset] [length]",
			Action: func(c *cli.Context) error {
				args := c.Args()
				if len(args) < 1 {
					return cli.NewExitError("Insufficient arguments", 1)
				}
				file := args[0]

				fileLen, err := fileLength(file)
				if err != nil {
					// TODO: Handle error
					return err
				}

				var offset int64
				if len(args) >= 2 {
					if offset, err = strconv.ParseInt(args[1], 0, 64); err != nil {
						return cli.NewExitError("Could not parse offset", 1)
					}
					if offset < 0 {
						return cli.NewExitError("offset cannot be before start of file", 1)
					}
					if offset >= fileLen {
						return cli.NewExitError("offset cannot be past end of file", 1)
					}
				}

				length := fileLen - offset
				if len(args) >= 3 {
					if length, err = strconv.ParseInt(args[2], 0, 64); err != nil {
						return cli.NewExitError("Could not parse length", 1)
					}
					if length < 0 {
						return cli.NewExitError("length cannot be negative", 1)
					}
					if length > fileLen {
						length = fileLen
					}
				}

				loadAddress = c.Int("loadaddr")
				return disasmFile(file, offset, length)
			},
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "loadaddr",
					Value: 0,
					Usage: "load address for the code",
				},
			},
		},
	}
	app.Run(os.Args)
}
