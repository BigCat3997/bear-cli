//go:build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

type iconEntry struct {
	typeCode string
	fileName string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: create_icns <iconset-directory> <output.icns>")
		os.Exit(2)
	}

	entries := []iconEntry{
		{typeCode: "icp4", fileName: "icon_16x16.png"},
		{typeCode: "icp5", fileName: "icon_32x32.png"},
		{typeCode: "icp6", fileName: "icon_32x32@2x.png"},
		{typeCode: "ic07", fileName: "icon_128x128.png"},
		{typeCode: "ic08", fileName: "icon_256x256.png"},
		{typeCode: "ic09", fileName: "icon_512x512.png"},
		{typeCode: "ic10", fileName: "icon_512x512@2x.png"},
	}

	type encodedEntry struct {
		typeCode string
		data     []byte
	}
	encoded := make([]encodedEntry, 0, len(entries))
	totalSize := uint32(8)
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(os.Args[1], entry.fileName))
		if err != nil {
			fatal(err)
		}
		encoded = append(encoded, encodedEntry{typeCode: entry.typeCode, data: data})
		totalSize += uint32(8 + len(data))
	}

	output, err := os.Create(os.Args[2])
	if err != nil {
		fatal(err)
	}
	defer output.Close()

	if _, err := output.WriteString("icns"); err != nil {
		fatal(err)
	}
	if err := binary.Write(output, binary.BigEndian, totalSize); err != nil {
		fatal(err)
	}
	for _, entry := range encoded {
		if _, err := output.WriteString(entry.typeCode); err != nil {
			fatal(err)
		}
		if err := binary.Write(output, binary.BigEndian, uint32(8+len(entry.data))); err != nil {
			fatal(err)
		}
		if _, err := output.Write(entry.data); err != nil {
			fatal(err)
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
