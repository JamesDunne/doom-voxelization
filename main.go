package main

import (
	"awesomeProject/matrix4"
	"awesomeProject/vector3"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
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
		pal = append(pal, color.RGBA{
			R: palLump.Data[i*3+0],
			G: palLump.Data[i*3+1],
			B: palLump.Data[i*3+2],
			A: 255,
		})
	}
	{
		col := pal[0xFF].(color.RGBA)
		col.A = 0
	}

	// Calculate camera angles and directions
	cameraTransforms := [8]matrix4.M{}
	for i := 0; i < 8; i++ {
		w := math.Pi * 2.0 * (float64(i) / 8.0)
		cameraTransforms[i] = matrix4.RotationZ(w)
	}
	// adjust near-zeros to zeros:
	cameraTransforms[0][1] = 0
	cameraTransforms[0][4] = 0
	cameraTransforms[2][0] = 0
	cameraTransforms[2][5] = 0
	cameraTransforms[4][1] = 0
	cameraTransforms[4][4] = 0
	cameraTransforms[6][0] = 0
	cameraTransforms[6][5] = 0

	//	for i := 0; i < 8; i++ {
	//		fmt.Printf(`
	//t[%d] = %25.22f %25.22f %25.22f %25.22f
	//       %25.22f %25.22f %25.22f %25.22f
	//       %25.22f %25.22f %25.22f %25.22f
	//       %25.22f %25.22f %25.22f %25.22f
	//`,
	//			i,
	//			cameraTransforms[i][0], cameraTransforms[i][1], cameraTransforms[i][2], cameraTransforms[i][3],
	//			cameraTransforms[i][4], cameraTransforms[i][5], cameraTransforms[i][6], cameraTransforms[i][7],
	//			cameraTransforms[i][8], cameraTransforms[i][9], cameraTransforms[i][10], cameraTransforms[i][11],
	//			cameraTransforms[i][12], cameraTransforms[i][13], cameraTransforms[i][14], cameraTransforms[i][15],
	//		)
	//	}

	//cameraTransforms[1] = cameraTransforms[1].Multiply(matrix4.RotationZ(math.Pi * 5.0 / 360.0))
	//cameraTransforms[7] = cameraTransforms[7].Multiply(matrix4.RotationZ(math.Pi * -5.0 / 360.0))

	//cameraTransforms[3] = cameraTransforms[3].Multiply(matrix4.RotationZ(math.Pi * -5.0 / 360.0))
	//cameraTransforms[5] = cameraTransforms[5].Multiply(matrix4.RotationZ(math.Pi * 5.0 / 360.0))

	reorder := []int{
		1,
		3,
		5,
		7,
		2,
		6,
		4,
		0,
	}

	//baseNames := []string{"CYBR", "SPID", "VILE", "POSS", "SPOS", "CPOS", "TROO", "SARG"}
	baseNames := []string{"CYBR", "SPID"}
	//baseNames := []string{"CYBR"}
	//baseNames := []string{"SPID"}

	// left, top:
	postAdj := make(map[string]*[8][2]int)

	postAdj["CYBRA"] = &[8][2]int{}
	postAdj["CYBRA"][0] = [2]int{0, -1}
	postAdj["CYBRA"][1] = [2]int{0, -1}
	postAdj["CYBRA"][2] = [2]int{0, -2}
	postAdj["CYBRA"][3] = [2]int{0, -5}
	postAdj["CYBRA"][4] = [2]int{0, -5}
	postAdj["CYBRA"][5] = [2]int{0, -4}
	postAdj["CYBRA"][6] = [2]int{0, -4}
	postAdj["CYBRA"][7] = [2]int{0, -4}

	postAdj["SPIDA"] = &[8][2]int{}
	postAdj["SPIDA"][0] = [2]int{0, 0}
	postAdj["SPIDA"][1] = [2]int{0, 0}
	postAdj["SPIDA"][2] = [2]int{0, 3}
	postAdj["SPIDA"][3] = [2]int{0, 3}
	postAdj["SPIDA"][4] = [2]int{0, 7}
	postAdj["SPIDA"][5] = [2]int{0, 2}
	postAdj["SPIDA"][6] = [2]int{0, 2}
	postAdj["SPIDA"][7] = [2]int{0, 2}

	for _, baseName := range baseNames {
		for frame := 0; frame < 5; frame++ {
			frameCh := uint8('A' + frame)
			baseFrameLumpName := fmt.Sprintf("%s%c", baseName, frameCh)

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
					}
				}

				// break when 8 found:
				return lumpsFound == 8
			})

			xmin := 256
			ymin := 256
			xmax := 0
			ymax := 0

			rotations := [8]*image.Paletted{}
			for p, lump := range lumps {
				spr := lump.Data
				size := uint32(len(spr))

				width := int(le.Uint16(spr[0:2]))
				height := int(le.Uint16(spr[2:4]))
				leftoffs := int(int16(le.Uint16(spr[4:6])))
				topoffs := int(int16(le.Uint16(spr[6:8])))
				_, _, _ = leftoffs, topoffs, height

				rect := image.Rect(0, 0, 256, 256)
				img := image.NewPaletted(rect, pal)
				for i := range img.Pix {
					img.Pix[i] = 0xFF
				}

				if adj, ok := postAdj[baseFrameLumpName]; ok && adj != nil {
					leftoffs += adj[p][0]
					topoffs += adj[p][1]
				}

				fmt.Printf("%s%d: (%d, %d)\n", baseFrameLumpName, p+1, leftoffs, topoffs)

				xadj := 256/2 - leftoffs
				yadj := 256 - 16 - topoffs

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
							x := xadj + width - 1 - i
							if x < xmin {
								xmin = x
							}
							if x > xmax {
								xmax = x
							}
							for j := 3; j < 3+length; j++ {
								y := yadj + ystart + j - 3
								if y < ymin {
									ymin = y
								}
								if y > ymax {
									ymax = y
								}
								img.SetColorIndex(x, y, run[j])
							}
						} else {
							x := xadj + i
							if x < xmin {
								xmin = x
							}
							if x > xmax {
								xmax = x
							}
							for j := 3; j < 3+length; j++ {
								y := yadj + ystart + j - 3
								if y < ymin {
									ymin = y
								}
								if y > ymax {
									ymax = y
								}
								img.SetColorIndex(x, y, run[j])
							}
						}
						runOffs += uint32(length)
						runOffs++
					}
				}

				rotations[p] = img

				func() {
					f, err := os.Create(fmt.Sprintf("fr-%s%c%d.png", baseName, frameCh, p+1))
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

			fmt.Printf("%d, %d, %d, %d\n", xmin, ymin, xmax, ymax)
			for p, img := range rotations {
				// extract minimal image:
				rotations[p] = image.NewPaletted(image.Rect(0, 0, xmax-xmin, ymax-ymin), img.Palette)
				draw.Draw(rotations[p], rotations[p].Rect, img, image.Point{X: xmin, Y: ymin}, draw.Over)
			}

			maxwidth := xmax - xmin
			maxheight := ymax - ymin

			maxx := 256
			maxy := 256
			maxz := 256

			// Create a voxel volume
			// X - (width)
			// Y / (depth)
			// Z | (height)
			voxels := make([][][]uint8, maxx)
			for i := 0; i < maxx; i++ {
				voxels[i] = make([][]uint8, maxy)
				for j := 0; j < maxy; j++ {
					voxels[i][j] = make([]uint8, maxz)
				}
			}

			volume := make([][][]bool, maxx)
			for i := 0; i < maxx; i++ {
				volume[i] = make([][]bool, maxy)
				for j := 0; j < maxy; j++ {
					volume[i][j] = make([]bool, maxz)
				}
			}

			horizCenter := float64(maxwidth) / 2.0
			vertCenter := float64(maxheight) / 2.0

			xCenter := float64(maxx) / 2.0
			yCenter := float64(maxy) / 2.0
			zCenter := float64(maxz) / 2.0

			const step = 4
			radius := float64(maxwidth) * math.Cos(0.25*math.Pi)
			halfRadius := radius / 2.0

			if true {
				fmt.Printf("mdl-%s%c.vox: voxelize step 1/3\n", baseName, frameCh)

				if true {
					for i := range reorder {
						img := rotations[reorder[i]]

						for u := 0; u < maxwidth; u++ {
							for v := 0; v < maxheight; v++ {
								c := img.ColorIndexAt(u, maxheight-1-v)
								if c != 0xFF {
									for t := 0.0; t < radius*step; t++ {
										p := vector3.V{
											X: float64(u) - horizCenter,
											Y: float64(t)*(1.0/step) - halfRadius,
											Z: float64(v) - vertCenter,
										}
										p = cameraTransforms[reorder[i]].Transform(p)

										x := int(math.Round(p.X + xCenter))
										if x < 0 || x >= maxx {
											continue
										}
										y := int(math.Round(p.Y + yCenter))
										if y < 0 || y >= maxy {
											continue
										}
										z := int(math.Round(p.Z + zCenter))
										if z < 0 || z >= maxz {
											continue
										}

										volume[x][y][z] = true
										voxels[x][y][z] = c
									}
								}
							}
						}
					}
				} else {
					for x := 0; x < maxwidth; x++ {
						for y := 0; y < maxwidth; y++ {
							for z := 0; z < maxheight; z++ {
								volume[x][y][z] = true
							}
						}
					}
				}

				if true {
					fmt.Printf("mdl-%s%c.vox: voxelize step 2/3\n", baseName, frameCh)
					for i := range reorder {
						img := rotations[reorder[i]]

						for u := 0; u < maxwidth; u++ {
							for v := 0; v < maxheight; v++ {
								c := img.ColorIndexAt(u, maxheight-1-v)
								if c == 0xFF {
									for t := 0.0; t < radius*step; t++ {
										p := vector3.V{
											X: float64(u) - horizCenter,
											Y: float64(t)*(1.0/step) - halfRadius,
											Z: float64(v) - vertCenter,
										}
										p = cameraTransforms[reorder[i]].Transform(p)

										x := int(math.Round(p.X + xCenter))
										if x < 0 || x >= maxx {
											continue
										}
										y := int(math.Round(p.Y + yCenter))
										if y < 0 || y >= maxy {
											continue
										}
										z := int(math.Round(p.Z + zCenter))
										if z < 0 || z >= maxz {
											continue
										}

										volume[x][y][z] = false
									}
								}
							}
						}

						// wipe out anything outside the image bounds:
						for u := maxwidth; u < int(float64(maxwidth)*1.4142136); u++ {
							for v := 0; v < maxheight; v++ {
								for t := 0.0; t < radius*step; t++ {
									p := vector3.V{
										X: float64(u) - horizCenter,
										Y: float64(t)*(1.0/step) - halfRadius,
										Z: float64(v) - vertCenter,
									}
									p = cameraTransforms[reorder[i]].Transform(p)

									x := int(math.Round(p.X + xCenter))
									if x < 0 || x >= maxx {
										continue
									}
									y := int(math.Round(p.Y + yCenter))
									if y < 0 || y >= maxy {
										continue
									}
									z := int(math.Round(p.Z + zCenter))
									if z < 0 || z >= maxz {
										continue
									}

									volume[x][y][z] = false
								}

								for t := 0.0; t < radius*step; t++ {
									p := vector3.V{
										X: float64(maxwidth-u) - horizCenter,
										Y: float64(t)*(1.0/step) - halfRadius,
										Z: float64(v) - vertCenter,
									}
									p = cameraTransforms[reorder[i]].Transform(p)

									x := int(math.Round(p.X + xCenter))
									if x < 0 || x >= maxx {
										continue
									}
									y := int(math.Round(p.Y + yCenter))
									if y < 0 || y >= maxy {
										continue
									}
									z := int(math.Round(p.Z + zCenter))
									if z < 0 || z >= maxz {
										continue
									}

									volume[x][y][z] = false
								}
							}
						}
					}
				}

				// recolor the surfaces from each angle:
				if true {
					fmt.Printf("mdl-%s%c.vox: voxelize step 3/3\n", baseName, frameCh)
					for i := range reorder {
						img := rotations[reorder[i]]
						for u := 0; u < maxwidth; u++ {
							for v := 0; v < maxheight; v++ {
								c := img.ColorIndexAt(u, maxheight-1-v)
								if c != 0xFF {
									for t := 0.0; t < radius*step; t++ {
										p := vector3.V{
											X: float64(u) - horizCenter,
											Y: float64(t)*(1.0/step) - halfRadius,
											Z: float64(v) - vertCenter,
										}
										p = cameraTransforms[reorder[i]].Transform(p)

										x := int(math.Round(p.X + xCenter))
										if x < 0 || x >= maxx {
											continue
										}
										y := int(math.Round(p.Y + yCenter))
										if y < 0 || y >= maxy {
											continue
										}
										z := int(math.Round(p.Z + zCenter))
										if z < 0 || z >= maxz {
											continue
										}

										voxels[x][y][z] = c

										// only color the surface:
										if volume[x][y][z] {
											break
										}
									}
								}
							}
						}
					}
				}
			} else {
				// projection and rotation test:
				angles := []int{0, 1, 2, 3, 4, 5, 6, 7}
				for _, i := range angles {
					img := rotations[i]

					for u := 0; u < maxwidth; u++ {
						for v := 0; v < maxheight; v++ {
							c := img.ColorIndexAt(u, maxheight-1-v)
							if c != 0xFF {
								p := vector3.V{
									X: float64(u) - horizCenter,
									Y: -halfRadius,
									Z: float64(v) - vertCenter,
								}
								p = cameraTransforms[i].Transform(p)

								x := int(math.Round(p.X + xCenter))
								if x < 0 || x >= maxx {
									continue
								}
								y := int(math.Round(p.Y + yCenter))
								if y < 0 || y >= maxy {
									continue
								}
								z := int(math.Round(p.Z + zCenter))
								if z < 0 || z >= maxz {
									continue
								}

								volume[x][y][z] = true
								voxels[x][y][z] = c
							}
						}
					}
				}
			}

			fmt.Printf("mdl-%s%c.vox: saving...\n", baseName, frameCh)
			err = saveVoxel(
				os.ExpandEnv(
					fmt.Sprintf("$HOME/Downloads/MagicaVoxel-0.99.6.2-macos-10.15/vox/mdl-%s%c.vox", baseName, frameCh),
				),
				uint32(maxx),
				uint32(maxy),
				uint32(maxz),
				pal,
				voxels,
				volume,
			)
			fmt.Printf("mdl-%s%c.vox: saved\n", baseName, frameCh)
		}
	}
}

func saveVoxel(voxPath string, maxx, maxy, maxz uint32, pal color.Palette, voxels [][][]byte, volume [][][]bool) (err error) {
	// Create VOX output file
	file := &bytes.Buffer{}

	// Write VOX header with version
	if _, err = file.Write([]byte("VOX ")); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, uint32(150)); err != nil {
		return
	}

	// Write MAIN chunk to describe the size of the file
	if _, err = file.Write([]byte("MAIN")); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, uint32(0)); err != nil {
		return
	}

	var mainLenOffs = file.Len()
	if err = binary.Write(file, binary.LittleEndian, uint32(0)); err != nil {
		return
	}

	// Write SIZE chunk to describe the voxel dimensions
	if _, err = file.Write([]byte("SIZE")); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, uint32(0xC)); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, uint32(0)); err != nil {
		return
	}
	// x
	if err = binary.Write(file, binary.LittleEndian, maxx); err != nil {
		return
	}
	// y
	if err = binary.Write(file, binary.LittleEndian, maxy); err != nil {
		return
	}
	// z
	if err = binary.Write(file, binary.LittleEndian, maxz); err != nil {
		return
	}

	// Write XYZI voxel data for indexed-color voxels at X,Y,Z locations
	if _, err = file.Write([]byte("XYZI")); err != nil {
		return
	}

	var offsSize = file.Len()

	size := uint32(4)
	count := uint32(0)
	if err = binary.Write(file, binary.LittleEndian, size); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, uint32(0)); err != nil {
		return
	}

	var offsCount = file.Len()
	if err = binary.Write(file, binary.LittleEndian, count); err != nil {
		return
	}

	for x := uint32(0); x < maxx; x++ {
		for y := uint32(0); y < maxy; y++ {
			for z := uint32(0); z < maxz; z++ {
				if volume[x][y][z] {
					c := voxels[x][y][z]
					if _, err = file.Write([]byte{byte(x), byte(y), byte(z), c + 1}); err != nil {
						return
					}
					count++
					size += 4
				}
			}
		}
	}

	// Write palette
	if _, err = file.Write([]byte("RGBA")); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, uint32(0x400)); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, uint32(0)); err != nil {
		return
	}
	for _, c := range pal {
		rgba := c.(color.RGBA)
		if _, err = file.Write([]byte{
			rgba.R,
			rgba.G,
			rgba.B,
			rgba.A,
		}); err != nil {
			return
		}
	}

	// calculate total file size and rewrite MAIN chunk:
	var fileSize int
	fileSize = file.Len()
	mainSize := uint32(fileSize) - 0x14

	// capture the buffer:
	b := file.Bytes()

	// patch over the buffer to fill in MAIN size:
	binary.LittleEndian.PutUint32(b[mainLenOffs:mainLenOffs+4], mainSize)

	// rewrite XYZI header:
	binary.LittleEndian.PutUint32(b[offsSize:offsSize+4], size)
	binary.LittleEndian.PutUint32(b[offsCount:offsCount+4], count)

	// write the file:
	if err = os.WriteFile(voxPath, b, 0644); err != nil {
		return
	}

	return
}
