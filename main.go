package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
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

	// find palette:
	palLump := wc.FindLumpBetween("PLAYPAL", "", "")
	if palLump == nil {
		panic("could not find PLAYPAL")
	}
	// load palette:
	pal := make(color.Palette, 0, 256)
	for i := 0; i < 256; i++ {
		pal = append(pal, color.NRGBA{
			R: palLump.Data[i*3+0],
			G: palLump.Data[i*3+1],
			B: palLump.Data[i*3+2],
			A: 255,
		})
	}

	{
		lump := wc.FindLumpBetween("TROOA2C8", "S_START", "S_END")
		if lump == nil {
			panic("could not find lump")
		}

		spr := lump.Data
		size := uint32(len(spr))

		{
			width := int(le.Uint16(spr[0:2]))
			height := int(le.Uint16(spr[2:4]))
			leftoffs := int16(le.Uint16(spr[4:6]))
			topoffs := int16(le.Uint16(spr[6:8]))
			_, _ = leftoffs, topoffs

			img := image.NewPaletted(image.Rect(0, 0, width, height), pal)

			for i := 0; i < width; i++ {
				runOffs := le.Uint32(spr[8+i*4 : 8+i*4+4])

			posts:
				for runOffs < size {
					run := spr[runOffs:size]

					ystart := int(run[0])
					if ystart == 0xff {
						break posts
					}

					//fmt.Printf("\033[%dG", ystart*3+1)
					//for j := 0; j < ystart; j++ {
					//	fmt.Printf("   ")
					//}

					length := int(run[1])
					// skip unused byte
					runOffs += 3
					for j := 3; j < 3+length; j++ {
						img.SetColorIndex(i, ystart+j-3, run[j])
						//fmt.Printf("%02x ", run[j])
					}
					runOffs += uint32(length)
					runOffs++
				}
				//fmt.Print("\n")
			}

			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					c := img.ColorIndexAt(x, y)
					if c != 0 {
						fmt.Printf("%02x ", c)
					} else {
						fmt.Print("   ")
					}
				}
				fmt.Print("\n")
			}

			func() {
				f, err := os.OpenFile("troo.png", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
				if err != nil {
					panic(err)
				}
				defer f.Close()
				err = png.Encode(f, img)
				if err != nil {
					panic(err)
				}
			}()
		}
	}
}
