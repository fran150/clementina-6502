// Package assets embeds static data files that ship with the emulator so the
// binary does not depend on the current working directory (which breaks under
// `go test`) or on external files being present at runtime.
package assets

import (
	"embed"
	"fmt"
)

// MiaKernel is the Clementina MIA bootstrap ROM image
// (assets/computer/mia/kernel.bin), embedded into the binary at build time.
//
// It is bound at build time, so editing kernel.bin is picked up on the next
// build (including `go run`, which rebuilds each invocation).
//
//go:embed computer/mia/kernel.bin
var MiaKernel []byte

// miaCharsetsFS holds the selectable MIA character sets, already in MIA pixel
// order so they copy straight into MIA RAM. videoLoadDefaultFont accepts two
// layouts (see there):
//   - Plane-0 blocks: a sequence of 2048-byte blocks, block i into plane 0 of
//     CHR bank i (e.g. openroms.bin = text + graphics, 4096 bytes).
//   - Full CHR dump: a nonzero multiple of a full 6144-byte CHR bank (3 planes),
//     loaded flat into the CHR region (e.g. clascii.bin = all 8 banks, 49152
//     bytes, exported by tools/tile-editor.html).
//
// They are produced offline (tile editor or the firmware repo's
// scripts/generate_charset.py) and kept in sync between repos.
//
//go:embed computer/mia/charsets/*.bin
var miaCharsetsFS embed.FS

// MiaCharset returns the CHR bank-0 image for the named character set (e.g.
// "openroms"), as embedded under computer/mia/charsets/<name>.bin.
func MiaCharset(name string) ([]byte, error) {
	data, err := miaCharsetsFS.ReadFile("computer/mia/charsets/" + name + ".bin")
	if err != nil {
		return nil, fmt.Errorf("unknown charset %q: %w", name, err)
	}
	return data, nil
}

// miaPalettesFS holds selectable MIA default palettes, in the tile editor's
// Palette (.bin) export format: 16 banks * 8 little-endian RGB565 colors.
//
//go:embed computer/mia/palettes/*.palette.bin
var miaPalettesFS embed.FS

// MiaPalette returns the default video palette for the named palette set (e.g.
// "clementina-text"), as embedded under computer/mia/palettes/<name>.palette.bin.
func MiaPalette(name string) ([]byte, error) {
	data, err := miaPalettesFS.ReadFile("computer/mia/palettes/" + name + ".palette.bin")
	if err != nil {
		return nil, fmt.Errorf("unknown palette %q: %w", name, err)
	}
	return data, nil
}
