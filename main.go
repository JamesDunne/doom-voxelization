package main

import (
	"fmt"
	"os"
)

var wc WADCollection

func main() {
	var err error

	iwadPath := os.ExpandEnv("$DOOMWADDIR/DOOM2.WAD")
	err = wc.Load(iwadPath)
	if err != nil {
		panic(err)
	}

	pwadPath := os.ExpandEnv("$DOOMWADDIR/D2SPFX20.WAD")
	err = wc.Load(pwadPath)
	if err != nil {
		panic(err)
	}

	for _, wad := range wc.Ordered {
		for _, lump := range wad.Lumps {
			fmt.Printf("%-12s\t%-8s\t%x\n", wad.Name, lump.Name, len(lump.Data))
		}
	}

	{
		lump := wc.FindLumpBetween("TROOA2C8", "S_START", "S_END")
		if lump == nil {
			panic("could not find lump")
		}

		spr := lump.Data
		size := uint32(len(spr))

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
