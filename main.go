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
	palLump := wc.FindLumpBetween("", "", func(s string) bool {
		if s == "PLAYPAL" {
			return true
		}
		return false
	})
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

	for _, baseName := range []string{"POSS", "SPOS", "CPOS", "TROO", "SARG", "CYBR", "SPID", "VILE"} {
		// find all 8 sprite rotations:
		lumps := [8]*Lump{}
		lumpsFound := 0
		wc.IterateLumpsBetween("S_START", "S_END", func(lump *Lump) bool {
			s := lump.Name
			if s[:4] != baseName {
				return false
			}

			if len(s) >= 6 {
				if s[4] == 'A' {
					//fmt.Printf("%s\n", s)
					fr := s[5] - '0'
					if fr >= 1 && fr <= 8 {
						if lumps[fr-1] == nil {
							lumps[fr-1] = lump
							lumpsFound++
						}
					}
					// break when 8 found:
					return lumpsFound == 8
				}
			}

			if len(s) == 8 {
				if s[6] == 'A' {
					//fmt.Printf("%s\n", s)
					fr := s[7] - '0'
					if fr >= 1 && fr <= 8 {
						if lumps[fr-1] == nil {
							lump.HFlip = true
							lumps[fr-1] = lump
							lumpsFound++
						}
					}
					// break when 8 found:
					return lumpsFound == 8
				}
			}

			return false
		})

		// find image extents of full rotation:
		maxwidth := 0
		maxheight := 0
		for p, lump := range lumps {
			if lump == nil {
				panic(fmt.Sprintf("could not find lump A%d", p+1))
			}
			spr := lump.Data

			width := int(le.Uint16(spr[0:2]))
			height := int(le.Uint16(spr[2:4]))
			leftoffs := int(int16(le.Uint16(spr[4:6])))
			topoffs := int(int16(le.Uint16(spr[6:8])))

			width += leftoffs
			height += 16
			if width > maxwidth {
				maxwidth = width
			}
			if height > maxheight {
				maxheight = height
			}
			_, _ = leftoffs, topoffs
		}

		for p, lump := range lumps {
			spr := lump.Data
			size := uint32(len(spr))

			{
				width := int(le.Uint16(spr[0:2]))
				height := int(le.Uint16(spr[2:4]))
				leftoffs := int(int16(le.Uint16(spr[4:6])))
				topoffs := int(int16(le.Uint16(spr[6:8])))
				_, _, _ = leftoffs, topoffs, height

				//fmt.Printf("%sA%d: %d, %d\n", baseName, p+1, leftoffs, topoffs)

				img := image.NewPaletted(image.Rect(0, 0, maxwidth, maxheight), pal)

				xadj := maxwidth/2 - leftoffs
				yadj := maxheight - 16 - topoffs

				for i := 0; i < width; i++ {
					runOffs := le.Uint32(spr[8+i*4 : 8+i*4+4])

				posts:
					for runOffs < size {
						run := spr[runOffs:size]

						ystart := int(run[0])
						if ystart == 0xff {
							break posts
						}

						length := int(run[1])
						// skip unused byte
						runOffs += 3
						if lump.HFlip {
							// h-flip:
							for j := 3; j < 3+length; j++ {
								img.SetColorIndex(xadj+width-1-i, yadj+ystart+j-3, run[j])
							}
						} else {
							for j := 3; j < 3+length; j++ {
								img.SetColorIndex(xadj+i, yadj+ystart+j-3, run[j])
							}
						}
						runOffs += uint32(length)
						runOffs++
					}
				}

				//for y := 0; y < height; y++ {
				//	for x := 0; x < width; x++ {
				//		c := img.ColorIndexAt(x, y)
				//		if c != 0 {
				//			fmt.Printf("%02x ", c)
				//		} else {
				//			fmt.Print("   ")
				//		}
				//	}
				//	fmt.Print("\n")
				//}

				func() {
					f, err := os.OpenFile(fmt.Sprintf("fr-%sA%d.png", baseName, p+1), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
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
}
