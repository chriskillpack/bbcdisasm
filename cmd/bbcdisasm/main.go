package main

import (
	"bbcdisasm"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/urfave/cli/v2"
)

func listDFS(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading %s\n", file)
		return err
	}

	img := bbcdisasm.ParseDFS(data)
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

func extractFromDfs(file string, entries []string, outDir string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

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

	// TODO: Replace with sort.Search when we have a function that handles !BOOT correctly
	// sort.SearchStrings has false positive for !BOOT
	em := make(map[string]bool)
	for _, entry := range entries {
		em[entry] = true
	}

	img := bbcdisasm.ParseDFS(data)
	for _, f := range img.Files {
		if len(entries) == 0 || em[f.Filename] {
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
	app.Name = "bbcdisasm"
	app.Usage = "Tool to extract and disassemble programs from BBC Micro DFS disk images"
	app.Action = func(c *cli.Context) error {
		cli.ShowAppHelp(c)
		return nil
	}

	app.Commands = []*cli.Command{
		{
			Name:      "list",
			Aliases:   []string{"ls"},
			Usage:     "List a DFS disk image",
			ArgsUsage: "image",
			Action: func(c *cli.Context) error {
				args := c.Args()
				if args.Len() < 1 {
					return cli.Exit("Insufficient arguments", 1)
				}
				return listDFS(args.First())
			},
		},
		{
			Name:      "extract",
			Aliases:   []string{"x"},
			Usage:     "Extract one or more files from DFS disk image",
			ArgsUsage: "[--outdir outDir] image [entry] [entry] ... [entry]",
			Action: func(c *cli.Context) error {
				args := c.Args()
				image := args.First()
				if image == "" {
					return cli.Exit("No image provided", 1)
				}

				if err := extractFromDfs(image, args.Tail(), c.String("outdir")); err != nil {
					return cli.Exit("Could not extract file from image", 1)
				}
				return nil
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "outdir",
					Value: ".",
					Usage: "output directory for extracted files",
				},
			},
		},
		{
			Name:      "disasm",
			Aliases:   []string{"d"},
			Usage:     "Disassemble a file",
			ArgsUsage: "file [offset] [length]",
			Action:    disasmCmd,
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:  "loadaddr",
					Usage: "load address for the code",
				},
				&cli.StringFlag{
					Name:  "codeaddrs",
					Usage: "locations of known code",
				},
				&cli.StringSliceFlag{
					Name:    "definevar",
					Usage:   "<variable>=<value>",
					Aliases: []string{"D"},
				},
			},
		},
	}
	app.Run(os.Args)
}
