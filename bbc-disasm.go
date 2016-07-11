package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"
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
}

func disassemble(program []uint8, maxBytes, offset uint) {
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
			fmt.Printf("loop%d:\n", targetIdx)
		}

		b := program[cursor]

		fmt.Printf("0x%04X: ", cursor)
		if op, ok := OpCodesMap[b]; ok {
			instructions := program[cursor : cursor+op.length]
			s := op.decode(instructions, cursor)
			fmt.Printf("%s %v\n", op.name, s)
			cursor += op.length
		} else {
			// Gracefully handle unrecognized opcodes
			fmt.Printf("0x%02X\n", b)
			cursor++
		}
	}
}

func listDfs(file string) error {
	if data, err := ioutil.ReadFile(file); err != nil {
		fmt.Printf("Error reading %s", file)
		return err
	} else {
		img := parseDfs(data)
		listImage(img)
	}

	return nil
}

// http://mdfs.net/Docs/Comp/Disk/Format/DFS
func parseDfs(dfs []byte) diskImage {
	image := diskImage{}

	image.title = strings.TrimRight(string(dfs[0:8])+string(dfs[0x100:0x103]), "")

	nFiles := int(dfs[0x105]) / 8

	image.nSectors = int(dfs[0x107]) + int(dfs[0x106]&3)*256
	image.bootOpt = int(dfs[0x106]&48) >> 4
	image.cycle = int(dfs[0x104])
	image.files = make([]catalog, nFiles)

	// Read file catalog entries
	for i := 0; i < nFiles; i++ {
		var offset int

		// Read out the filename
		offset = 0x008 + i*8
		image.files[i].filename = strings.TrimRight(string(dfs[offset:offset+6]), " ")
		image.files[i].dir = string(dfs[offset+7])

		// Read file info
		offset = 0x108 + i*8
		image.files[i].length = int(dfs[offset+4]) + int(dfs[offset+5])*256 + int(dfs[offset+6]&48)*4096
		image.files[i].loadAddr = int(dfs[offset+0]) + int(dfs[offset+1])*256 + int(dfs[offset+6]&12)*16384
		image.files[i].execAddr = int(dfs[offset+2]) + int(dfs[offset+3])*256 + int(dfs[offset+6]&192)*1024
		image.files[i].startSector = int(dfs[offset+7]) + int(dfs[offset+6]&3)*256
	}

	return image
}

func listImage(image diskImage) {
	fmt.Printf("Disk Title  %s\n", image.title)
	fmt.Printf("Num Files   %d\n", len(image.files))
	fmt.Printf("Num Sectors %d\n", image.nSectors)
	fmt.Printf("Boot Option %d\n", image.bootOpt)
	fmt.Printf("Disk Cycle  0x%0X\n\n", image.cycle)

	fmt.Println("Filename Length LoadAddr ExecAddr Sector")
	for i, _ := range image.files {
		file := &image.files[i]
		fmt.Printf("%-6s   %04X   %08X %08X %3d\n", file.filename, file.length, file.loadAddr, file.execAddr, file.startSector)
	}
}

func extractFromDfs(file, entry, outName string) error {
	var data []byte
	var err error

	if data, err = ioutil.ReadFile(file); err != nil {
		fmt.Printf("Error reading %s", file)
		return err
	}

	img := parseDfs(data)
	for _, f := range img.files {
		if f.filename == entry || entry == "" {
			// Retrieve data contents
			offset := f.startSector * 256
			d := data[offset:(offset + f.length)]

			var ofn string
			if outName == "" {
				ofn = f.filename
			} else {
				ofn = outName
			}
			if err := ioutil.WriteFile(ofn, d, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func disasmFile(file string, offset, length int64) error {
	var data []byte
	var err error

	if data, err = ioutil.ReadFile(file); err != nil {
		fmt.Printf("Error reading %s", file)
		return err
	}

	disassemble(data, uint(length), uint(offset))
	return nil
}

func fileLength(filename string) (int64, error) {
	var f *os.File
	var err error
	if f, err = os.Open(filename); err != nil {
		return 0, err
	}

	var fi os.FileInfo
	if fi, err = f.Stat(); err != nil {
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
			ArgsUsage: "image [entry] [outName]",
			Action: func(c *cli.Context) error {
				args := c.Args()
				if len(args) < 1 {
					return cli.NewExitError("Insufficient arguments", 1)
				}
				var entry, outName string
				if len(args) >= 2 {
					entry = args[1]
				}
				if len(args) >= 3 {
					outName = args[2]
				}
				if err := extractFromDfs(args[0], entry, outName); err != nil {
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

				var fileLen int64
				var err error
				if fileLen, err = fileLength(file); err != nil {
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

				return disasmFile(file, offset, length)
			},
		},
	}
	app.Run(os.Args)
}
