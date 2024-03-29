# BBC Disasm

A work in progress disassembler for 6502 programs and Acorn DFS disk image extractor

## Build

Requires a version of Go that supports modules.

```bash
$ go install ./cmd/bbcdisasm
```

## Usage

### List disk image contents
List the contents of a DFS image, in this case Exile one of my favorite BBC B games and an amazing technical achievement in 32Kb of RAM.

```
$ bbcdisasm list images/Exile.ssd
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
$ bbcdisasm extract images/Exile.ssd EXILE
```

Or we can extract all the files from an image, again into the current directory

```bash
$ bbcdisasm extract images/Exile.ssd
```

To extract only EXILE and ExileL to subdirectory `out`

```bash
$ bbcdisasm extract --outdir out images/Exile.ssd EXILE ExileL
```

### Disassemble a file

This is a simple 2-pass 6502 byte-code disassembler that uses light knowledge of the BBC Micro memory map to replace well known memory address with their names, e.g. `0xFFF7` is the `OSCLI` entry point. The disassembler output is compatible with beebasm. A primary goal of the disassembler is assembling the disassembler output should yield a result identical with the binary input to the disassembler.

Let's disassemble the first non-BASIC program in the Exile disk image, EXILE, starting from it's execution point. The output below shows labelled branch targets and identification of OS entry point addresses.

```
$ bbcdisasm disasm --loadaddr 0x3000 EXILE 0x1A10
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
.label_0
 CPY &0DBC              \ &4A3A CC BC 0D    ...
 BEQ label_1            \ &4A3D F0 03       ..
 STA &02A1,Y            \ &4A3F 99 A1 02    ...
 ...
```

The `--loadaddr` option instructs the disassembler to 'relocate' the program to a different memory address. This is to match the actual memory address DFS will place the file contents. TODO: Apply loadaddr to the execution address.

The `--codeaddrs` option takes a comma-seperated list of addresses that the disassembler should treat as code and ensure that they are not skipped during disassembly. This is helpful in cases where data bytes ahead of the addressed match multibyte opcodes that cause the disassembler to miss important addresses.

By default `disasm` will disassemble the entire file though this can be limited by the optional final length argument. The disassembler will complete disassembly of an instruction if it straddles the length. In the example below the disassembler processes 9 bytes even though only 8 were asked for, because the final instruction straddles the 8 byte boundary:

```
$ bbcdisasm d --loadaddr 0x3000 exile/EXILE 0x1A10 8
...
 LDA #&C8               \ &4A10 A9 C8       ..
 LDX #&03               \ &4A12 A2 03       ..
 LDY #&00               \ &4A14 A0 00       ..
 JSR OSBYTE             \ &4A16 20 F4 FF     ..
```

#### User defined variables

The disassembler allows simple variables to be declared via command line options in the form `-D <name>=<value>`. Some operand values are checked against these variables and on a match the literal value will be replaced with the variable name. The variable definitions are included at the top of the disassembly before code disassembly.

```
$ bbcdisasm d -D apples=0x0DBC --loadaddr 0x3000 exile_files/EXILE 0x1A38 16
...
\ Defined Variables
apples = 0x0DBC
...
 LDY #&0F               \ &4A38 A0 0F       ..
.label_0
 CPY apples             \ &4A3A CC BC 0D    ...
 BEQ label_1            \ &4A3D F0 03       ..
```

Without the variable definition the `CPY` line would be disassembled as

```
 CPY &0DBC              \ &4A3A CC BC 0D    ...
```

#### beebasm workaround

beebasm has a trait that need to be worked around, "zero page replacement". In this situation an instruction with an absolute address in the zero-page is replaced with the zero page form of the instruction, e.g. `LDA &0012` (`AD`, `12`, `00`) will be assembled as `LDA &12` (`A5`, `12`). This break binary compatibility. The disassembler will identify instructions where this will happen and emit instead as a data sequence `EQUB &AD, &12, &00`. This situation generally happens when disassembling data, as written code will prefer the zero page form as it is faster and uses less bytes.

#### Unknown instructions

It is very common for BBC micro programs to store data amongst code. The disassembler has no knowledge of where these blocks of data are so it will attempt to disassemble everything. Not all byte values map to 6502 instructions, in this case the byte value will be emitted as a 'data byte' using the `EQUB` directive:

```
 EQUB &6F               \ &3491 6F          o
```

#### Undocumented instructions

There is very limited support for undocumented instructions in the 6502. This is partly because beebasm, the targeted assembler, does not support them. In order to preserve binary compatibility `bbcdisasm` will emit the opcode bytes of the instruction as an `EQUB` directive and comment the line with `UD` (UnDocumented) together with a very limited instruction mnemonic.

The byte sequence `&53,&63` disassembles to `SRE (&63),Y`, an undocumented instruction, and will be output as
```
 EQUB &53,&63           \ UD SRE            Sc
```

## TODO

* Improved BBC Micro memory map support in the disassembler
* User supplied variable list