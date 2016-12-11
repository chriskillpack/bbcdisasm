# BBC Disasm

A work in progress disassembler for 6502 programs and Acorn DFS disk image extractor

## Build

```bash
$ go get github.com/urfave/cli
$ go build
```

## Usage

### List disk image contents
List the contents of a DFS image, in this case Exile one of my favorite BBC B games and an amazing technical achievement in 32Kb of RAM.

```bash
$ bbc-disasm list images/Exile.ssd
Disk Title  EXILE
Num Files   7
Num Sectors 800
Boot Option 3
Disk Cycle  0x10

Filename Length LoadAddr ExecAddr Sector
LOAD     0103   00031900 00038023 316
!BOOT    000E   00000000 0003FFFF 315
ExileS   102D   00031900 00031900 298
ExileM   6570   00031200 00037690 196
ExileL   45AA   00033000 000374E0 126
ExileB   6080   00031200 00037200  29
EXILE    1A80   00033000 00034A10   2
```

### Extract a file from the disk image

Let's extract the EXILE program from the Exile.ssd image saving it in the current directory as EXILE

```bash
$ bbc-disasm extract images/Exile.ssd EXILE
```

Or we can extract all the files from the image to the current directory

```bash
$ bbc-disasm extract images/Exile.ssd
```

To extract all files to subdirectory `out`

```bash
$ bbc-disasm extract images/Exile.ssd "" out
```

### Disassemble a file

This is a simple 2-pass 6502 byte-code disassembler that has light knowledge of the BBC Micro memory map and will mark up some absolute memory address, e.g. that `0xFFF7` is the `OSCLI` entry point.

Let's disassemble the first non-BASIC program in the Exile disk image, EXILE, starting from it's execution point. The output below shows loop targets and identification of OS entry point addresses.

```bash
$ bbc-disasm disasm --loadaddr 0x3000 EXILE 0x1A00
loop0:
0x4A00: LDA $4948,Y
0x4A03: BMI +8  (loop1,0x4A0B)
0x4A05: JSR $FFEE  (OSWRCH)
0x4A08: INY
0x4A09: BNE -9  (loop0,0x4A00)
loop1:
0x4A0B: RTS
0x4A0C: BRK
0x4A0D: BRK
0x4A0E: BRK
0x4A0F: BRK
0x4A10: LDA #$C8
0x4A12: LDX #$03
0x4A14: LDY #$00
0x4A16: JSR $FFF4  (OSBYTE)
...
0x4A7E: LSR $54
```

The `--loadaddr` options instructs the disassembler to 'relocate' the program to a different memory address. This is to match the actual memory address DFS will place the file contents. TODO: Apply loadaddr to the execution address.

By default `disasm` will disassemble the entire file though this can be limited by the optional final length argument

```bash
$ bbc-disasm d --loadaddr 0x3000 exile/EXILE 0x1A00 8
0x4A00: LDA $4948,Y
0x4A03: BMI +8  (loop0,0x4A0B)
0x4A05: JSR $FFEE  (OSWRCH)
```

## TODO

* Cleaner disassembler output
* Improved BBC Micro memory map support in the disassembler
