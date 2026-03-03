package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

// Format constants
const (
	FormatUnknown     = ""
	FormatMcWorld     = ".mcworld"
	FormatLitematic   = ".litematic"
	FormatSchem       = ".schem"
	FormatNbt         = ".nbt"
	FormatMcStructure = ".mcstructure"
)

// DetectFormat handles the stream-based format detection.
// It uses Peek for magic numbers and a lightweight NBT scanner for internal structure.
func DetectFormat(data []byte) (string, error) {
	size := int64(len(data))
	r := bytes.NewReader(data)
	
	// 進化3: 爆弾対策 (1MBまでしかスキャンしない)
	limitedReader := io.LimitReader(r, 1024*1024)
	bufReader := bufio.NewReader(limitedReader)

	magic, err := bufReader.Peek(4)
	if err != nil {
		if err == io.EOF && len(data) > 0 {
			// small file, continue with what we have
		} else {
			return FormatUnknown, err
		}
	}

	// 1. Zip判定 (.mcworld)
	if isZip(magic) {
		if hasLevelDatInZip(data, size) {
			return FormatMcWorld, nil
		}
		return FormatUnknown, nil
	}

	// 2. 圧縮解除後の NBT 構造チェック
	var nbtReader io.ReadCloser
	if isGzip(magic) {
		gz, err := gzip.NewReader(bufReader)
		if err != nil {
			return FormatUnknown, nil
		}
		nbtReader = gz
	} else if isZlib(magic) {
		zl, err := zlib.NewReader(bufReader)
		if err != nil {
			return FormatUnknown, nil
		}
		nbtReader = zl
	} else {
		// 非圧縮 NBT (または未知の形式)
		nbtReader = io.NopCloser(bufReader)
	}
	defer nbtReader.Close()

	// 抽出したルートキーで判定
	keys, rootName, err := scanRootNBTKeys(nbtReader)
	if err != nil {
		return FormatUnknown, nil
	}

	// 判定ロジック (Detectorパターン)
	if rootName == "Schematic" {
		return FormatSchem, nil
	}
	
	if keys["Metadata"] && keys["Regions"] {
		return FormatLitematic, nil
	}
	
	if keys["size"] && keys["blocks"] && keys["palette"] {
		return FormatNbt, nil
	}
	
	if keys["structure"] && keys["format_version"] {
		return FormatMcStructure, nil
	}

	return FormatUnknown, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func isZip(magic []byte) bool {
	return len(magic) >= 4 && magic[0] == 0x50 && magic[1] == 0x4B && magic[2] == 0x03 && magic[3] == 0x04
}

func isGzip(magic []byte) bool {
	return len(magic) >= 2 && magic[0] == 0x1F && magic[1] == 0x8B
}

func isZlib(magic []byte) bool {
	if len(magic) < 2 {
		return false
	}
	return magic[0] == 0x78 && (magic[1] == 0x01 || magic[1] == 0x9C || magic[1] == 0xDA)
}

// hasLevelDatInZip checks if level.dat exists in the ZIP archive without full extraction.
func hasLevelDatInZip(data []byte, size int64) bool {
	zr, err := zip.NewReader(bytes.NewReader(data), size)
	if err != nil {
		return false
	}
	for _, f := range zr.File {
		if f.Name == "level.dat" || f.Name == "db/" {
			return true
		}
	}
	return false
}

// scanRootNBTKeys is a lightweight SAX-style scanner.
// It only reads the root compound's primary children names.
func scanRootNBTKeys(r io.Reader) (map[string]bool, string, error) {
	keys := make(map[string]bool)
	
	// Read Tag ID (must be Tag_Compound = 10)
	var tagID byte
	if err := binary.Read(r, binary.BigEndian, &tagID); err != nil {
		return nil, "", err
	}
	if tagID != 10 {
		return nil, "", fmt.Errorf("not a compound tag")
	}

	// Read Root Name
	rootName, err := readNBTString(r)
	if err != nil {
		return nil, "", err
	}

	// Iterate through children
	for {
		var childID byte
		if err := binary.Read(r, binary.BigEndian, &childID); err != nil {
			break // likely end of stream or limited reader hit
		}
		if childID == 0 { // Tag_End
			break
		}

		name, err := readNBTString(r)
		if err != nil {
			break
		}
		keys[name] = true

		// Skip the payload of this child
		if err := skipNBTPayload(r, childID); err != nil {
			break
		}
		
		// 性能向上のため、必要な形式が揃った時点で早期終了も可能だか、
		// 判定キーは数十個程度なので全走査でも十分速い。
	}

	return keys, rootName, nil
}

func readNBTString(r io.Reader) (string, error) {
	var length uint16
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return "", err
	}
	if length == 0 {
		return "", nil
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// skipNBTPayload advances the reader past the payload of the given NBT tag type.
func skipNBTPayload(r io.Reader, tagID byte) error {
	switch tagID {
	case 1: // Byte
		return discard(r, 1)
	case 2: // Short
		return discard(r, 2)
	case 3: // Int
		return discard(r, 4)
	case 4: // Long
		return discard(r, 8)
	case 5: // Float
		return discard(r, 4)
	case 6: // Double
		return discard(r, 8)
	case 7: // Byte Array
		var length int32
		binary.Read(r, binary.BigEndian, &length)
		return discard(r, int64(length))
	case 8: // String
		var length uint16
		binary.Read(r, binary.BigEndian, &length)
		return discard(r, int64(length))
	case 9: // List
		var subType byte
		var length int32
		binary.Read(r, binary.BigEndian, &subType)
		binary.Read(r, binary.BigEndian, &length)
		for i := 0; i < int(length); i++ {
			if err := skipNBTPayload(r, subType); err != nil {
				return err
			}
		}
		return nil
	case 10: // Compound
		for {
			var subID byte
			if err := binary.Read(r, binary.BigEndian, &subID); err != nil {
				return err
			}
			if subID == 0 {
				break
			}
			readNBTString(r) // name
			if err := skipNBTPayload(r, subID); err != nil {
				return err
			}
		}
		return nil
	case 11: // Int Array
		var length int32
		binary.Read(r, binary.BigEndian, &length)
		return discard(r, int64(length)*4)
	case 12: // Long Array
		var length int32
		binary.Read(r, binary.BigEndian, &length)
		return discard(r, int64(length)*8)
	}
	return nil
}

func discard(r io.Reader, n int64) error {
	if n <= 0 {
		return nil
	}
	_, err := io.CopyN(io.Discard, r, n)
	return err
}
