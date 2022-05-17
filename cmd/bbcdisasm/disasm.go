package main

import (
	"bbcdisasm"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

func disasmCmd(c *cli.Context) error {
	args := c.Args()
	if args.Len() < 1 {
		return cli.Exit("Insufficient arguments", 1)
	}
	file := args.First()

	fileLen, err := fileLength(file)
	if err != nil {
		return cli.Exit(err, 1)
	}

	// Is there an offset from program start for disassembly to begin?
	var offset int64
	if args.Len() >= 2 {
		if offset, err = strconv.ParseInt(args.Get(1), 0, 64); err != nil {
			return cli.Exit("Could not parse offset", 1)
		}
		if offset < 0 {
			return cli.Exit("offset cannot be before start of file", 1)
		}
		if offset >= fileLen {
			return cli.Exit("offset cannot be past end of file", 1)
		}
	}

	// Is there an optional length argument?
	length := fileLen - offset
	if args.Len() >= 3 {
		if length, err = strconv.ParseInt(args.Get(2), 0, 64); err != nil {
			return cli.Exit("Could not parse length", 1)
		}
		if length < 0 {
			return cli.Exit("length cannot be negative", 1)
		}
		if length > fileLen {
			length = fileLen
		}
	}

	disasm, err := disassemblerForFile(file)
	if err != nil {
		return cli.Exit(err, 1)
	}
	disasm.MaxBytes = uint(length)
	disasm.Offset = uint(offset)
	disasm.BranchAdjust = uint(c.Int("loadaddr"))

	caddrs := c.String("codeaddrs")
	if len(caddrs) > 0 {
		saddrs := strings.Split(caddrs, ",")
		for _, addr := range saddrs {
			i, err := strconv.ParseInt(addr, 0, 64)
			if err != nil {
				return cli.Exit("Could not parse address", 1)
			}
			if i < 0 {
				return cli.Exit("Invalid address", 1)
			}
			disasm.CodeAddrs = append(disasm.CodeAddrs, uint(i))
		}
	}

	dvars := c.StringSlice("definevar")
	for _, dvar := range dvars {
		parts := strings.Split(dvar, "=")
		if len(parts) <= 1 {
			return cli.Exit(fmt.Sprintf("invalid variable definition %q", dvar), 1)
		}
		err := disasm.AddVar(parts[0], parts[1])
		if err != nil {
			return cli.Exit(fmt.Sprintf("non numeric value %q", parts[1]), 1)
		}
	}

	disasm.Disassemble(os.Stdout)
	return nil
}

func disassemblerForFile(file string) (*bbcdisasm.Disassembler, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return bbcdisasm.NewDisassembler(data), nil
}
