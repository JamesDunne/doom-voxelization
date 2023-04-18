package main

import (
	"encoding/binary"
	"fmt"
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
		pal = append(pal, color.RGBA{
			R: palLump.Data[i*3+0],
			G: palLump.Data[i*3+1],
			B: palLump.Data[i*3+2],
			A: 255,
		})
	}

	// Calculate camera angles and directions
	//cameraAngles := []float64{0, math.Pi / 4, math.Pi / 2, 3 * math.Pi / 4, math.Pi, 5 * math.Pi / 4, 3 * math.Pi / 2, 7 * math.Pi / 4}
	cameraDirections := make([][3]float64, 8)
	for i := 0; i < 8; i++ {
		w := (2.0 / 8.0) * math.Pi * float64((8-i+6)&7)
		cameraDirections[i] = [3]float64{
			math.Cos(w),
			-math.Sin(w),
			0,
		}

		fmt.Printf("c[%d] = {%7.3f, %7.3f, %7.3f}\n", i, cameraDirections[i][0], cameraDirections[i][1], cameraDirections[i][2])
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
			}

			// Create a voxel volume
			// X - (width)
			// Y / (depth)
			// Z | (height)
			voxels := make([][][]uint8, maxwidth)
			for i := 0; i < maxwidth; i++ {
				voxels[i] = make([][]uint8, maxwidth)
				for j := 0; j < maxwidth; j++ {
					voxels[i][j] = make([]uint8, maxheight)
				}
			}

			volume := make([][][]bool, maxwidth)
			for i := 0; i < maxwidth; i++ {
				volume[i] = make([][]bool, maxwidth)
				for j := 0; j < maxwidth; j++ {
					volume[i][j] = make([]bool, maxheight)
				}
			}

			horizCenter := float64(maxwidth) / 2
			vertCenter := float64(maxheight) / 2

			if false {
				// Iterate over each voxel in the volume
				modelImages := rotations[:1]

				for x := 0; x < maxwidth; x++ {
					for y := 0; y < maxwidth; y++ {
						for z := 0; z < maxheight; z++ {
							// Project voxel onto each image
							hitCount := 0
							bestColor := uint8(0)
							for i, img := range modelImages {
								cameraDirection := cameraDirections[i]
								rayStart := [3]float64{
									float64(x) - horizCenter,
									float64(y) - horizCenter,
									float64(z) - vertCenter,
								}

							imgScan:
								for u := 0; u < maxwidth; u++ {
									for v := 0; v < maxheight; v++ {
										c := img.ColorIndexAt(u, maxheight-1-v)
										if c > 0 {
											var possible bool
											var point [3]float64
											rayEnd := [3]float64{
												float64(u) - horizCenter,
												//0,
												0 - horizCenter,
												float64(v) - vertCenter,
											}
											//rayEnd = vcross(rayEnd, vneg(cameraDirection))

											possible, point = intersects(rayStart, rayEnd, cameraDirection)
											if !possible {
												continue
											}
											//if point[0] >= -horizCenter && point[0] < horizCenter &&
											//	point[2] >= -vertCenter && point[2] < vertCenter {
											_ = point
											{
												c := img.ColorIndexAt(u, maxheight-1-v)
												if c > 0 {
													// TODO: pick best color of voxel:
													if c > bestColor {
														bestColor = c
													}
													hitCount++
													break imgScan
												}
											}
										}

									}
								}

							}

							// Set voxel value based on number of hits
							if hitCount >= 1 {
								voxels[x][y][z] = bestColor
								volume[x][y][z] = true
							}
						}
					}
				}
			}

			// try trivial projection:
			if true {
				for u := 0; u < maxwidth; u++ {
					for v := 0; v < maxheight; v++ {
						for i, img := range rotations {
							c := img.ColorIndexAt(u, v)
							if c > 0 {
								p := [3]float64{
									float64(u) - horizCenter,
									0,
									float64(v) - vertCenter,
								}
								p[1] -= 32
								p = vrotzangle(p, 1.5*math.Pi)
								p = vrotzvec(p, cameraDirections[i])

								x, y, z := int(p[0]+horizCenter), int(p[1]+horizCenter), int(p[2]+vertCenter)
								if x < 0 || x >= maxwidth {
									continue
								}
								if y < 0 || y >= maxwidth {
									continue
								}
								if z < 0 || z >= maxheight {
									continue
								}
								volume[x][y][maxheight-1-z] = true
								voxels[x][y][maxheight-1-z] = c
							}
						}
					}
				}
			}

			err = saveVoxel(
				os.ExpandEnv(
					fmt.Sprintf("$HOME/Downloads/MagicaVoxel-0.99.6.2-macos-10.15/vox/mdl-%s%c.vox", baseName, frameCh),
				),
				uint32(maxwidth),
				uint32(maxwidth),
				uint32(maxheight),
				pal,
				voxels,
				volume,
			)
		}
	}
}

// Check if a ray intersects with a plane
func intersects(rayStart, rayEnd, planeNormal [3]float64) (possible bool, intersection [3]float64) {
	// Compute direction of ray
	dir := [3]float64{
		rayEnd[0] - rayStart[0],
		rayEnd[1] - rayStart[1],
		rayEnd[2] - rayStart[2],
	}

	// Check if ray is parallel to plane
	denom := vdot(planeNormal, dir)
	if denom == 0 {
		possible = false
		return
	}

	// Compute intersection point between ray and plane
	t := vdot(planeNormal, rayStart) / denom
	if t < 0 {
		possible = false
		return
	}

	intersection = [3]float64{
		rayStart[0] + t*dir[0],
		rayStart[1] + t*dir[1],
		rayStart[2] + t*dir[2],
	}

	possible = true
	return
}

// Dot product of two 3D vectors
func vdot(a, b [3]float64) float64 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}

func vcross(a, b [3]float64) [3]float64 {
	return [3]float64{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

func vmul(a [3]float64, s float64) [3]float64 {
	return [3]float64{
		a[0] * s,
		a[1] * s,
		a[2] * s,
	}
}

func vneg(a [3]float64) [3]float64 {
	return [3]float64{
		-a[0],
		-a[1],
		-a[2],
	}
}

func vrotzangle(a [3]float64, theta float64) [3]float64 {
	c, s := math.Cos(theta), math.Sin(theta)
	return [3]float64{
		a[0]*c - a[1]*s,
		a[0]*s + a[1]*c,
		a[2],
	}
}

func vrotzvec(a [3]float64, v [3]float64) [3]float64 {
	return [3]float64{
		a[0]*v[0] - a[1]*v[1],
		a[0]*v[1] + a[1]*v[0],
		a[2],
	}
}

func saveVoxel(voxPath string, maxx, maxy, maxz uint32, pal color.Palette, voxels [][][]byte, volume [][][]bool) (err error) {
	// Create VOX output file
	var file *os.File
	if file, err = os.Create(voxPath); err != nil {
		return
	}
	defer file.Close()

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

	var mainLenOffs int64
	if mainLenOffs, err = file.Seek(0, io.SeekCurrent); err != nil {
		return
	}
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

	var offsSize int64
	if offsSize, err = file.Seek(0, io.SeekCurrent); err != nil {
		return
	}

	size := uint32(4)
	count := uint32(0)
	if err = binary.Write(file, binary.LittleEndian, size); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, uint32(0)); err != nil {
		return
	}

	var offsCount int64
	if offsCount, err = file.Seek(0, io.SeekCurrent); err != nil {
		return
	}
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

	// rewrite XYZI header:
	if _, err = file.Seek(offsSize, io.SeekStart); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, size); err != nil {
		return
	}

	if _, err = file.Seek(offsCount, io.SeekStart); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, count); err != nil {
		return
	}

	// seek back to end:
	if _, err = file.Seek(0, io.SeekEnd); err != nil {
		return
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
	var fileSize int64
	if fileSize, err = file.Seek(0, io.SeekCurrent); err != nil {
		return
	}
	mainSize := uint32(fileSize) - 0x14

	if _, err = file.Seek(mainLenOffs, io.SeekStart); err != nil {
		return
	}
	if err = binary.Write(file, binary.LittleEndian, mainSize); err != nil {
		return
	}

	return
}
