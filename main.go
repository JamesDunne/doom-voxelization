package main

import (
	"encoding/binary"
	"fmt"
	"github.com/deeean/go-vector/vector3"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
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

	cameraPos := [8]vector3.Vector3{}
	for i := 0; i < 8; i++ {
		w := 2 * math.Pi * float64((i+6)&7) / 8.0
		cameraPos[i].Set(
			math.Cos(w)*64,
			0, //41+16,
			math.Sin(w)*64,
		)

		fmt.Printf("c[%d] = {%7.3f, %7.3f, %7.3f}\n", i, cameraPos[i].X, cameraPos[i].Y, cameraPos[i].Z)
	}

	//baseNames := []string{"POSS", "SPOS", "CPOS", "TROO", "SARG", "CYBR", "SPID", "VILE"}
	baseNames := []string{"POSS"}
	for _, baseName := range baseNames {
		frame := 0
		{
			frameCh := uint8('A' + frame)

			// find all 8 sprite rotations:
			lumps := [8]*Lump{}
			lumpsFound := 0
			wc.IterateLumpsBetween("S_START", "S_END", func(lump *Lump) bool {
				s := lump.Name
				if s[:4] != baseName {
					return false
				}

				if len(s) >= 6 {
					if s[4] == frameCh {
						r := s[5] - '0'
						if r >= 1 && r <= 8 {
							if lumps[r-1] == nil {
								lump.HFlip = false
								lumps[r-1] = lump
								lumpsFound++
							}
						}
						// break when 8 found:
						return lumpsFound == 8
					}
				}

				if len(s) == 8 {
					if s[6] == frameCh {
						r := s[7] - '0'
						if r >= 1 && r <= 8 {
							if lumps[r-1] == nil {
								lump.HFlip = true
								lumps[r-1] = lump
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
					panic(fmt.Sprintf("could not find lump %c%d", frameCh, p+1))
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

			rotations := [8]*image.Paletted{}
			for p, lump := range lumps {
				spr := lump.Data
				size := uint32(len(spr))

				{
					width := int(le.Uint16(spr[0:2]))
					height := int(le.Uint16(spr[2:4]))
					leftoffs := int(int16(le.Uint16(spr[4:6])))
					topoffs := int(int16(le.Uint16(spr[6:8])))
					_, _, _ = leftoffs, topoffs, height

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

					rotations[p] = img

					func() {
						f, err := os.OpenFile(fmt.Sprintf("fr-%s%c%d.png", baseName, frameCh, p+1), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
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

			// attempt to voxelize:
			voxels := make([]byte, maxwidth*maxheight*maxwidth)
			worldPos := vector3.Vector3{}
			for x := 0; x < maxwidth; x++ {
				worldPos.X = float64(x) - float64(maxwidth)/2.0
				for z := 0; z < maxwidth; z++ {
					worldPos.Z = float64(z) - float64(maxwidth)/2.0
					for y := 0; y < maxheight; y++ {
						worldPos.Y = float64(y) - float64(maxheight)/2.0

						for i := 0; i < 8; i++ {
							dir := worldPos.Sub(&cameraPos[i])
							dir.Normalize()

							px := int(float64(maxwidth) * (math.Atan2(dir.X, dir.Z)/(2*math.Pi) + 0.5))
							py := int(float64(maxheight) * (math.Atan2(dir.Y, math.Sqrt(dir.X*dir.X+dir.Z*dir.Z))/(1*math.Pi) + 0.5))

							if px < 0 || py < 0 {
								continue
							}
							if px >= maxwidth || py >= maxheight {
								continue
							}

							c := rotations[i].ColorIndexAt(px, py)
							if c > 0 {
								voxels[(z*maxwidth*maxheight)+y*maxwidth+x] = c
								break
							}
						}
					}
				}
			}

			func() {
				// Create output file
				file, err := os.Create(fmt.Sprintf("mdl-%s%c.vox", baseName, frameCh))
				if err != nil {
					panic(err)
				}
				defer file.Close()

				// Write header
				file.Write([]byte("VOX "))
				binary.Write(file, binary.LittleEndian, uint32(150))

				// Write main chunk
				file.Write([]byte("MAIN"))
				binary.Write(file, binary.LittleEndian, uint32(0))
				binary.Write(file, binary.LittleEndian, uint32(0))

				file.Write([]byte("SIZE"))
				binary.Write(file, binary.LittleEndian, uint32(0xC))
				binary.Write(file, binary.LittleEndian, uint32(0))
				binary.Write(file, binary.LittleEndian, uint32(maxwidth))  // x
				binary.Write(file, binary.LittleEndian, uint32(maxheight)) // y
				binary.Write(file, binary.LittleEndian, uint32(maxwidth))  // z

				// Write voxel data
				file.Write([]byte("XYZI"))
				size := uint32(4)
				count := uint32(0)
				binary.Write(file, binary.LittleEndian, size)
				binary.Write(file, binary.LittleEndian, uint32(0))
				binary.Write(file, binary.LittleEndian, count)
				for x := 0; x < maxwidth; x++ {
					for y := 0; y < maxheight; y++ {
						for z := 0; z < maxwidth; z++ {
							c := voxels[(z*maxwidth*maxheight)+y*maxwidth+x]
							if c != 0 {
								file.Write([]byte{byte(x), byte(y), byte(z), c})
								count++
								size += 4
							}
						}
					}
				}

				// rewrite XYZI header:
				file.Seek(12*4, io.SeekStart)
				binary.Write(file, binary.LittleEndian, size)
				binary.Write(file, binary.LittleEndian, uint32(0))
				binary.Write(file, binary.LittleEndian, count)

				file.Seek(0, io.SeekEnd)

				// Write palette
				file.Write([]byte("RGBA"))
				binary.Write(file, binary.LittleEndian, uint32(0x400))
				binary.Write(file, binary.LittleEndian, uint32(0))
				for _, c := range pal {
					rgba := c.(color.NRGBA)
					file.Write([]byte{
						rgba.R,
						rgba.G,
						rgba.B,
						rgba.A,
					})
				}

				var fi os.FileInfo
				fi, err = file.Stat()
				mainSize := uint32(fi.Size()) - 0x14

				file.Seek(4*4, io.SeekStart)
				binary.Write(file, binary.LittleEndian, mainSize)
			}()
		}
	}
}
