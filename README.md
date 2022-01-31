# BBC Disasm

A work in progress disassembler for 6502 programs and Acorn DFS disk image extractor

## Build

[Requires a version of Go that supports modules]

```bash
$ go build .
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

Filename  Length LoadAddr ExecAddr Sector
LOAD      0103   00031900 00038023 316
!BOOT     000E   00000000 0003FFFF 315
ExileSR   102D   00031900 00031900 298
ExileMC   6570   00031200 00037690 196
ExileL    45AA   00033000 000374E0 126
ExileB    6080   00031200 00037200  29
EXILE     1A80   00033000 00034A10   2
```

### Extract file(s) from the disk image

Let's extract EXILE program from the Exile.ssd image into the current directory

```bash
$ bbc-disasm extract images/Exile.ssd EXILE
```

Or we can extract all the files from an image, again into the current directory

```bash
$ bbc-disasm extract images/Exile.ssd
```

To extract only EXILE and ExileL to subdirectory `out`

```bash
$ bbc-disasm extract --outdir out images/Exile.ssd EXILE ExileL
```

### Disassemble a file

This is a simple 2-pass 6502 byte-code disassembler that uses light knowledge of the BBC Micro memory map to replace well known memory address with their names, e.g. `0xFFF7` is the `OSCLI` entry point.

Let's disassemble the first non-BASIC program in the Exile disk image, EXILE, starting from it's execution point. The output below shows loop targets and identification of OS entry point addresses.

```
$ bbc-disasm disasm --loadaddr 0x3000 EXILE 0x1A10
...
CODE% = &3000

 LDA #&C8               \ &4A10 A9 C8       ..
 LDX #&03               \ &4A12 A2 03       ..
 LDY #&00               \ &4A14 A0 00       ..
 JSR OSBYTE             \ &4A16 20 F4 FF     ..
 LDY #&00               \ &4A19 A0 00       ..
 JSR &4A00              \ &4A1B 20 00 4A     .J
 JSR &4980              \ &4A1E 20 80 49     .I
 LDY #&28               \ &4A21 A0 28       .(
 JSR &4A00              \ &4A23 20 00 4A     .J
 LDA #&15               \ &4A26 A9 15       ..
 LDX #&00               \ &4A28 A2 00       ..
 JSR OSBYTE             \ &4A2A 20 F4 FF     ..
 LDA #&81               \ &4A2D A9 81       ..
 LDX #&20               \ &4A2F A2 20       .
 LDY #&03               \ &4A31 A0 03       ..
 JSR OSBYTE             \ &4A33 20 F4 FF     ..
 LDA #&00               \ &4A36 A9 00       ..
 LDY #&0F               \ &4A38 A0 0F       ..
.loop_0
 CPY &0DBC              \ &4A3A CC BC 0D    ...
 BEQ loop_1             \ &4A3D F0 03       ..
 STA &02A1,Y            \ &4A3F 99 A1 02    ...
 ...
```

The `--loadaddr` options instructs the disassembler to 'relocate' the program to a different memory address. This is to match the actual memory address DFS will place the file contents. TODO: Apply loadaddr to the execution address.

By default `disasm` will disassemble the entire file though this can be limited by the optional final length argument

```
$ bbc-disasm d --loadaddr 0x3000 exile/EXILE 0x1A10 8
...
 LDA #&C8               \ &4A10 A9 C8       ..
 LDX #&03               \ &4A12 A2 03       ..
 LDY #&00               \ &4A14 A0 00       ..
 JSR OSBYTE             \ &4A16 20 F4 FF     ..
```

#### Undocumented instructions

There is very limited support for undocumented instructions in the 6502. This is partly because beebasm, the targeted assembler, does not support them. In order to preserve binary compatibility `bbc-disasm` will emit the opcode bytes of the instruction as an `EQUB` statement and comment the line with `UD` (UnDocumented) together with a very limited instruction mnemonic.

The byte sequence `&53,&63` disassembles to `SRE (&63),Y`, an undocumented instruction, and will be output as
```
 EQUB &53,&63           \ UD SRE            Sc
```

## TODO

* Improved BBC Micro memory map support in the disassembler
