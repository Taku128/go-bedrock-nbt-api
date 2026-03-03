package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"

	"github.com/Tnze/go-mc/nbt"
	"github.com/uberswe/mcnbt"
)

// ---------------------------------------------------------------------------
// ListTag encodes an []int32 as TAG_List(TAG_Int) instead of TAG_Int_Array.
// Java Edition's structure NBT strictly requires TAG_List for pos / size.
// ---------------------------------------------------------------------------

// ListTag is a custom type that serialises as TAG_List of TAG_Int in NBT.
type ListTag []int32

// TagType satisfies go-mc's nbt.Marshaler interface hint.
func (l ListTag) TagType() byte { return 9 } // TAG_List

// MarshalNBT writes the list header (element-type + length) followed by each int32.
func (l ListTag) MarshalNBT(w io.Writer) error {
	var buf [5]byte
	buf[0] = 3 // TAG_Int
	buf[1] = byte(len(l) >> 24)
	buf[2] = byte(len(l) >> 16)
	buf[3] = byte(len(l) >> 8)
	buf[4] = byte(len(l))
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	for _, v := range l {
		var vBuf [4]byte
		vBuf[0] = byte(v >> 24)
		vBuf[1] = byte(v >> 16)
		vBuf[2] = byte(v >> 8)
		vBuf[3] = byte(v)
		if _, err := w.Write(vBuf[:]); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Java NBT encoding structs
// ---------------------------------------------------------------------------

// PaletteEntry is a single block-state in the Java structure palette.
type PaletteEntry struct {
	Name       string            `nbt:"Name"`
	Properties map[string]string `nbt:"Properties,omitempty"`
}

// JavaBlock is a single placed block in Java structure format.
type JavaBlock struct {
	Pos   ListTag     `nbt:"pos"`
	State int32       `nbt:"state"`
	Nbt   interface{} `nbt:"nbt,omitempty"`
}

// JavaNBT is the top-level Java Edition structure NBT compound.
type JavaNBT struct {
	DataVersion int32          `nbt:"DataVersion"`
	Size        ListTag        `nbt:"size"`
	Palette     []PaletteEntry `nbt:"palette"`
	Blocks      []JavaBlock    `nbt:"blocks"`
	Entities    []interface{}  `nbt:"entities"`
}

// ---------------------------------------------------------------------------
// convertJavaNbt parses a Java schematic file (.litematic / .schem / .nbt)
// through the mcnbt library, converts it to StandardFormat, then re-encodes
// it as a gzip-compressed Java Edition structure NBT.
// ---------------------------------------------------------------------------
func convertJavaNbt(tmpFile string) ([]byte, error) {
	// 1. Parse input
	schemData, errParse := mcnbt.ParseAnyFromFileAsJSON(tmpFile)
	if errParse != nil {
		return nil, fmt.Errorf("failed to parse java schematic format: %w", errParse)
	}

	// 2. Convert to standard internal structure
	stdnbt, errConv := mcnbt.ConvertToStandard(schemData)
	if errConv != nil {
		return nil, fmt.Errorf("failed to standardize java format: %w", errConv)
	}

	// 3. Build Java NBT
	jNbt := JavaNBT{
		DataVersion: int32(stdnbt.DataVersion),
		Size:        ListTag{int32(stdnbt.Size.X), int32(stdnbt.Size.Y), int32(stdnbt.Size.Z)},
		Entities:    make([]interface{}, 0),
	}

	// Ensure no zero-sized axis
	if jNbt.Size[0] == 0 { jNbt.Size[0] = 1 }
	if jNbt.Size[1] == 0 { jNbt.Size[1] = 1 }
	if jNbt.Size[2] == 0 { jNbt.Size[2] = 1 }

	// 4. Build palette (sparse map → dense slice)
	maxPalette := -1
	for k := range stdnbt.Palette {
		if k > maxPalette {
			maxPalette = k
		}
	}
	if maxPalette >= 0 {
		jNbt.Palette = make([]PaletteEntry, maxPalette+1)
		for k, v := range stdnbt.Palette {
			jNbt.Palette[k] = PaletteEntry{
				Name:       v.Name,
				Properties: v.Properties,
			}
		}
	} else {
		jNbt.Palette = []PaletteEntry{{Name: "minecraft:air"}}
	}

	// 5. Build blocks (relative to Position origin)
	jNbt.Blocks = make([]JavaBlock, 0, len(stdnbt.Blocks))
	for _, b := range stdnbt.Blocks {
		if b.Type == "block" || b.Type == "" {
			jNbt.Blocks = append(jNbt.Blocks, JavaBlock{
				Pos: ListTag{
					int32(b.Position.X) - int32(stdnbt.Position.X),
					int32(b.Position.Y) - int32(stdnbt.Position.Y),
					int32(b.Position.Z) - int32(stdnbt.Position.Z),
				},
				State: int32(b.State),
				Nbt:   b.NBT,
			})
		}
	}

	log.Printf("convertJavaNbt stats: DataVersion=%d, Size=%v, Palette=%d, Blocks=%d",
		jNbt.DataVersion, jNbt.Size, len(jNbt.Palette), len(jNbt.Blocks))

	// 6. Encode to gzip-compressed NBT
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if err := nbt.NewEncoder(gw).Encode(jNbt, ""); err != nil {
		gw.Close()
		return nil, fmt.Errorf("failed to encode standardized NBT payload: %w", err)
	}
	gw.Close()

	return buf.Bytes(), nil
}
