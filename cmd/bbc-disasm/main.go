package main

import (
	bbc "bbc-disasm"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/urfave/cli"
)

var (
	loadAddress = 0
)

func listDFS(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading %s\n", file)
		return err
	}

	img := bbc.ParseDFS(data)
	fmt.Printf("Disk Title  %s\n", img.Title)
	fmt.Printf("Num Files   %d\n", len(img.Files))
	fmt.Printf("Num Sectors %d\n", img.Sectors)
	fmt.Printf("Boot Option %d\n", img.BootOpt)
	fmt.Printf("Disk Cycle  0x%0X\n\n", img.Cycle)

	fmt.Println("Filename  Length LoadAddr ExecAddr Sector")
	for _, file := range img.Files {
		fmt.Printf("%-7s   %04X   %08X %08X %3d\n", file.Filename, file.Length, file.LoadAddr, file.ExecAddr, file.StartSector)
	}

	return nil
}

func disasmFile(file string, offset, length int64, loadAddress uint) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading %s", file)
		return err
	}

	bbc.Disassemble(data, uint(length), uint(offset), loadAddress)
	return nil
}

func extractFromDfs(file, entry, outDir string) error {
	var data []byte
	var err error

	if data, err = ioutil.ReadFile(file); err != nil {
		fmt.Printf("Error reading %s\n", file)
		return err
	}

	img := bbc.ParseDFS(data)

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

	for _, f := range img.Files {
		if f.Filename == entry || entry == "" {
			// Retrieve data contents
			offset := f.StartSector * 256
			d := data[offset:(offset + f.Length)]

			ofn := path.Join(outDir, f.Filename)
			if err := ioutil.WriteFile(ofn, d, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func fileLength(filename string) (int64, error) {
	fi, err := os.Stat(filename)
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
				return listDFS(c.Args().First())
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

				loadAddress := c.Int("loadaddr")
				return disasmFile(file, offset, length, uint(loadAddress))
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
