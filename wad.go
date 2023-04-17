package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
)

var le = binary.LittleEndian

type Lump struct {
	Name string
	Data []byte

	HFlip bool
}

type WAD struct {
	Name       string
	AllData    []byte
	Lumps      []Lump
	LumpByName map[string]uint32
}

func (wad *WAD) Load(wadPath string) (err error) {
	var b []byte
	b, err = os.ReadFile(wadPath)
	if err != nil {
		return
	}

	numEntries := le.Uint32(b[4:8])
	directoryOffs := le.Uint32(b[8:12])

	wad.Name = filepath.Base(wadPath)
	wad.AllData = b
	wad.Lumps = make([]Lump, numEntries)
	wad.LumpByName = make(map[string]uint32, numEntries)

	for i, o := uint32(0), directoryOffs; i < numEntries; i, o = i+1, o+16 {
		offs := le.Uint32(b[o : o+4])
		size := le.Uint32(b[o+4 : o+8])
		name := b[o+8 : o+16]

		nameStr := string(bytes.TrimRight(name[:], "\x00"))

		wad.Lumps[i].Name = nameStr
		wad.Lumps[i].Data = b[offs : offs+size]

		wad.LumpByName[nameStr] = i
	}

	return
}

func (wad *WAD) FindLumpIndex(name string) uint32 {
	return wad.LumpByName[name]
}

func (wad *WAD) EndIndex() uint32 {
	return uint32(len(wad.Lumps) - 1)
}

func (wad *WAD) FindLumpBetween(start uint32, end uint32, matches func(string) bool) *Lump {
	absEnd := wad.EndIndex()
	if end > absEnd {
		end = absEnd
	}

	for i := start; i <= end; i++ {
		if matches(wad.Lumps[i].Name) {
			return &wad.Lumps[i]
		}
	}

	return nil
}

func (wad *WAD) IterateLumpsBetween(start uint32, end uint32, iter func(*Lump) bool) bool {
	absEnd := wad.EndIndex()
	if end > absEnd {
		end = absEnd
	}

	for i := start; i <= end; i++ {
		if iter(&wad.Lumps[i]) {
			// break:
			return true
		}
	}

	return false
}

type WADCollection struct {
	Ordered []*WAD
}

func (wc *WADCollection) Load(path string) (err error) {
	wad := &WAD{}
	err = wad.Load(path)
	if err != nil {
		return
	}

	// prepend to list:
	wc.Ordered = append([]*WAD{wad}, wc.Ordered...)

	return
}

func (wc *WADCollection) FindLumpBetween(start string, end string, matches func(string) bool) (lump *Lump) {
	for _, wad := range wc.Ordered {
		var startIndex uint32
		if start == "" {
			startIndex = 0
		} else {
			startIndex = wad.FindLumpIndex(start)
		}

		var endIndex uint32
		if end == "" {
			endIndex = wad.EndIndex()
		} else {
			endIndex = wad.FindLumpIndex(end)
		}

		lump = wad.FindLumpBetween(startIndex, endIndex, matches)
		if lump != nil {
			return
		}
	}

	return nil
}

func (wc *WADCollection) IterateLumpsBetween(start string, end string, iter func(*Lump) bool) {
	for _, wad := range wc.Ordered {
		var startIndex uint32
		if start == "" {
			startIndex = 0
		} else {
			startIndex = wad.FindLumpIndex(start)
		}

		var endIndex uint32
		if end == "" {
			endIndex = wad.EndIndex()
		} else {
			endIndex = wad.FindLumpIndex(end)
		}

		if wad.IterateLumpsBetween(startIndex, endIndex, iter) {
			break
		}
	}
}
