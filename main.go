package main

import (
  "flag"
  "fmt"
  "io/ioutil"
  "os"
  "strings"
)

func disassemble(program []uint8, maxBytes, offset uint) {
  var cursor uint = offset

  for cursor < (offset + maxBytes) {
    b := program[cursor]

    // Find the opcode
    var j int
    for j = 0 ; j < len(OpCodes) ; j++ {
      if (OpCodes[j].base == b) {
        break
      }
    }
    fmt.Printf("0x%04X: ", cursor)
    if j < len(OpCodes) {
      opcode := OpCodes[j]
      instructions := program[cursor:cursor + opcode.length]
      s := opcode.decodeFn(instructions, cursor)
      fmt.Printf("%s %v\n", opcode.name, s)
      cursor += opcode.length
    } else {
      fmt.Printf("0x%02X\n", b)
      cursor++
    }
  }
}

type diskImage struct {
  title string;
  nSectors int;
  bootOpt int;
  cycle int;
  files []catalog;
}

type catalog struct {
  filename string;
  dir string;
  length int;
  loadAddr int;
  execAddr int;
  startSector int;
}

// http://mdfs.net/Docs/Comp/Disk/Format/DFS
func parseDFS(dfs []byte) diskImage {
  image := diskImage{}

  image.title = strings.TrimRight(string(dfs[0:8]) + string(dfs[0x100:0x103]), "")

  nFiles := int(dfs[0x105]) / 8

  image.nSectors = int(dfs[0x107]) + int(dfs[0x106] & 3) * 256;
  image.bootOpt = int(dfs[0x106] & 48) >> 4
  image.cycle = int(dfs[0x104])
  image.files = make([]catalog, nFiles)

  // Read file catalog entries
  for i := 0 ; i < nFiles ; i++ {
    var offset int

    // Read out the filename
    offset = 8 + i * 8
    image.files[i].filename = strings.TrimRight(string(dfs[offset:offset+6]), " ")
    image.files[i].dir = string(dfs[offset+7])

    // Read file info
    offset = 0x108 + i * 8
    image.files[i].length = int(dfs[offset+4]) + int(dfs[offset+5]) * 256 + int(dfs[offset+6] & 48) * 4096
    image.files[i].loadAddr = int(dfs[offset+0]) + int(dfs[offset+1]) * 256 + int(dfs[offset+6] & 12) * 16384
    image.files[i].execAddr = int(dfs[offset+2]) + int(dfs[offset+3]) * 256 + int(dfs[offset+6] & 192) * 1024
    image.files[i].startSector = int(dfs[offset+7]) + int(dfs[offset+6] & 3) * 256
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

func extractFileFromImage(image diskImage, bytes []byte, filename string) error {
  var file *catalog = nil

  for _, entry := range image.files {
    if entry.filename == filename {
      file = &entry
      break
    }
  }
  if file == nil {
    // TODO: Return an error object
    return nil
  }

  offset := file.startSector * 256
  return ioutil.WriteFile("blahblah", bytes[offset:offset+file.length], 0644)
}

func main() {
  listPtr := flag.Bool("list", false, "List DFS filesystem and quit")
  // TODO: Flag to extract a particular file
  flag.Parse()

  data, _ := ioutil.ReadFile("Exile.ssd");
  // fileSize := len(data);

  diskImage := parseDFS(data)
  if (*listPtr) {
    listImage(diskImage)
    os.Exit(0)
  }

  // err := extractFileFromImage(diskImage, data, "EXILE")
  // if err != nil {
  //   os.Exit(1)
  // }

  // Disassemble beginning of exile program
  disassemble(data, 112, 7184)
  // Util func 4A00
  // disassemble(data, 16, 7168)
  // Util func 4980
  // disassemble(data, 128, 7040)

  // var offset uint = 512;
  // disassemble(data, uint(32), offset)

  // HexDump of file
  // numLines := fileSize / 16;
  // for i := 0 ; i < numLines + 1 ; i++ {
  //   numBytes := i * 16;
  //   if numBytes >= fileSize {
  //     numBytes = fileSize % 16;
  //   } else {
  //     numBytes = 16;
  //   }

  //   fmt.Printf("%04X  ", i * 16);
  //   for j := 0 ; j < numBytes ; j++ {
  //     index := i * 16 + j;
  //     fmt.Printf("%02x ", data[index]);
  //   }
  //   for j := 0 ; i < 16 - numBytes ; j++ {
  //     fmt.Print("   ");
  //   }
  //   fmt.Print("| ");
  //   for j := 0 ; j < numBytes ; j++ {
  //     index := i * 16 + j;
  //     opcode := data[index];

  //     if ((opcode >= 'A' && opcode <= 'Z') || (opcode >= 'a' && opcode <= 'z') ||
  //         (opcode >= '0' && opcode <= '9')) {
  //       fmt.Printf("%c", opcode);
  //     } else {
  //       fmt.Print(".");
  //     }
  //   }
  //   for j := 0 ; j < 16 - numBytes ; j++ {
  //     fmt.Print(' ');
  //   }
  //   fmt.Print(" |\n");
  // }
}
