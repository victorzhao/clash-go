// iconres writes the Windows .syso resource files that embed the app icon
// into the executable, so a plain `go build` produces an exe with the right
// icon in Explorer and the taskbar. The files are named after the target
// architecture and picked up by the Go linker automatically.
//
//	go run ./cmd/iconres
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/victorzhao/clash-go/internal/app"
)

var sizes = []int{16, 24, 32, 48, 64, 128, 256}

var arches = map[string]struct{ machine, relocation uint16 }{
	"amd64": {0x8664, 0x0003},
	"arm64": {0xAA64, 0x0002},
	"386":   {0x014C, 0x0007},
}

func main() {
	for name, arch := range arches {
		syso, err := build(arch.machine, arch.relocation)
		if err != nil {
			fmt.Fprintln(os.Stderr, name+":", err)
			os.Exit(1)
		}
		path := "mygo_windows_" + name + ".syso"
		if err := os.WriteFile(path, syso, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
	}
}

// build creates a COFF object with a .rsrc section holding RT_ICON and
// RT_GROUP_ICON resources from the app icon.
func build(machine, relocation uint16) ([]byte, error) {
	images := make([][]byte, len(sizes))
	for i, s := range sizes {
		images[i] = app.IconPNG(s)
		if images[i] == nil {
			return nil, fmt.Errorf("icon %d failed", s)
		}
	}

	// RT_GROUP_ICON: a directory pointing at each RT_ICON by id.
	var group bytes.Buffer
	_ = binary.Write(&group, binary.LittleEndian, [3]uint16{0, 1, uint16(len(images))})
	for i, img := range images {
		id := uint16(i + 1)
		dim := byte(sizes[i])
		if sizes[i] >= 256 {
			dim = 0
		}
		group.Write([]byte{dim, dim, 0, 0})
		_ = binary.Write(&group, binary.LittleEndian, [2]uint16{1, 32})
		_ = binary.Write(&group, binary.LittleEndian, uint32(len(img)))
		_ = binary.Write(&group, binary.LittleEndian, id)
	}

	// The resource tree has two types (icon, group icon), one id each,
	// one language each.
	const rtIcon, rtGroupIcon = 3, 14
	res := map[uint16]map[uint16][]byte{
		rtIcon:      {1: images[0], 2: images[1], 3: images[2], 4: images[3], 5: images[4], 6: images[5], 7: images[6]},
		rtGroupIcon: {1: group.Bytes()},
	}
	const lang = 0x0409 // en-US

	dirSize := func(n int) int { return 16 + 8*n }
	types := sortedKeys(res)
	size := dirSize(len(types))
	for _, t := range types {
		size += dirSize(len(res[t])) + len(res[t])*dirSize(1)
	}
	entriesAt := size
	count := 0
	for _, t := range types {
		count += len(res[t])
	}
	dataAt := entriesAt + 16*count

	raw := make([]byte, dataAt)
	le := binary.LittleEndian
	dir := func(at, n int) { le.PutUint16(raw[at+14:], uint16(n)) }
	dir(0, len(types))
	next := dirSize(len(types))
	entry := entriesAt
	var relocs []uint32
	var data bytes.Buffer
	for i, t := range types {
		le.PutUint32(raw[16+8*i:], uint32(t))
		le.PutUint32(raw[16+8*i+4:], uint32(next)|0x80000000)
		ids := sortedKeys(res[t])
		idDir := next
		dir(idDir, len(ids))
		next += dirSize(len(ids))
		for j, id := range ids {
			le.PutUint32(raw[idDir+16+8*j:], uint32(id))
			le.PutUint32(raw[idDir+16+8*j+4:], uint32(next)|0x80000000)
			dir(next, 1)
			le.PutUint32(raw[next+16:], lang)
			le.PutUint32(raw[next+20:], uint32(entry))
			next += dirSize(1)

			for data.Len()%8 != 0 {
				data.WriteByte(0)
			}
			le.PutUint32(raw[entry:], uint32(dataAt+data.Len()))
			le.PutUint32(raw[entry+4:], uint32(len(res[t][id])))
			relocs = append(relocs, uint32(entry))
			entry += 16
			data.Write(res[t][id])
		}
	}
	raw = append(raw, data.Bytes()...)
	for len(raw)%4 != 0 {
		raw = append(raw, 0)
	}

	var out bytes.Buffer
	const headerSize, sectionHeaderSize = 20, 40
	rawAt := headerSize + sectionHeaderSize
	relocAt := rawAt + len(raw)
	symbolsAt := relocAt + 10*len(relocs)
	_ = binary.Write(&out, le, struct {
		Machine, Sections                   uint16
		Time, Symbols, SymbolCount          uint32
		OptionalHeaderSize, Characteristics uint16
	}{machine, 1, 0, uint32(symbolsAt), 1, 0, 0})
	_ = binary.Write(&out, le, struct {
		Name                                        [8]byte
		VirtualSize, VirtualAddress, RawSize, RawAt uint32
		RelocAt, LineNumbersAt                      uint32
		Relocs, LineNumbers                         uint16
		Characteristics                             uint32
	}{[8]byte{'.', 'r', 's', 'r', 'c'}, 0, 0, uint32(len(raw)), uint32(rawAt), uint32(relocAt), 0, uint16(len(relocs)), 0, 0x40000040})
	out.Write(raw)
	for _, r := range relocs {
		_ = binary.Write(&out, le, r)
		_ = binary.Write(&out, le, uint32(0))
		_ = binary.Write(&out, le, relocation)
	}
	out.Write([]byte{'.', 'r', 's', 'r', 'c', 0, 0, 0})
	_ = binary.Write(&out, le, uint32(0))
	_ = binary.Write(&out, le, uint16(1))
	_ = binary.Write(&out, le, uint16(0))
	out.Write([]byte{3, 0})
	_ = binary.Write(&out, le, uint32(4))
	return out.Bytes(), nil
}

func sortedKeys[V any](m map[uint16]V) []uint16 {
	keys := make([]uint16, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
