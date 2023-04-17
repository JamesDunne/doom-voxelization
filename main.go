package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

var le = binary.LittleEndian

var wad []byte
var sprStart, sprEnd uint32

func findLump(name string) (ok bool, offs, size uint32) {
	nameLen := uint32(len(name))
	if nameLen > 8 {
		nameLen = 8
	}

	for o := sprStart; o < sprEnd; o = o + 16 {
		offs = le.Uint32(wad[o : o+4])
		size = le.Uint32(wad[o+4 : o+8])
		if bytes.Compare([]byte(name), wad[o+8:o+8+nameLen]) == 0 {
			ok = true
			return
		}
	}

	return
}

func main() {
	var err error

	path := os.ExpandEnv("$DOOMWADDIR/doom2.wad")
	wad, err = os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	//os.Stdout.WriteString(hex.Dump(wad[0:12]))

	numEntries := le.Uint32(wad[4:8])
	directoryOffs := le.Uint32(wad[8:12])

	//os.Stdout.WriteString(hex.Dump(wad[directoryOffs : directoryOffs+256]))

	for i, o := uint32(0), directoryOffs; i < numEntries; i, o = i+1, o+16 {
		name := wad[o+8 : o+16]
		if bytes.Compare([]byte("S_START\x00"), name) == 0 {
			sprStart = o + 16
		}
		if bytes.Compare([]byte("S_END\x00\x00\x00"), name) == 0 {
			sprEnd = o
			break
		}
	}

	if sprStart == 0 {
		return
	}

	for o := sprStart; o < sprEnd; o = o + 16 {
		offs := le.Uint32(wad[o : o+4])
		size := le.Uint32(wad[o+4 : o+8])
		fmt.Printf("%s\t%x\t%x\n", bytes.ReplaceAll(wad[o+8:o+16], []byte{0}, []byte{' '}), offs, size)
	}

	{
		ok, offs, size := findLump("TROOA1")
		if !ok {
			panic("could not find TROOA1")
		}
		fmt.Printf("%s\t%x\t%x\n", "TROOA1  ", offs, size)

		//os.Stdout.WriteString(hex.Dump(wad[offs : offs+256]))

		spr := wad[offs : offs+size]

		{
			runCount := le.Uint16(spr[0:2])

			for i := 0; i < int(runCount); i++ {
				runOffs := le.Uint32(spr[8+i*4 : 8+i*4+4])

			posts:
				for runOffs < size {
					run := spr[runOffs:size]

					ystart := int(run[0])
					if ystart == 0xff {
						break posts
					}

					fmt.Printf("\033[%dG", ystart*3+1)
					//for j := 0; j < ystart; j++ {
					//	fmt.Printf("   ")
					//}

					length := int(run[1])
					// skip unused byte
					runOffs += 3
					for j := 3; j < 3+length; j++ {
						fmt.Printf("%02x ", run[j])
					}
					runOffs += uint32(length)
					runOffs++
				}
				fmt.Print("\n")
			}
		}
	}
}
