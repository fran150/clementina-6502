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

// miaCharsetsFS holds the selectable MIA character sets. Each file is a flat
// CHR bank-0 image, already in MIA pixel order, so it can be copied straight
// into MIA RAM. A CHR bank is 6144 bytes (3 planes of 2048): plane 0
// (0x000..0x7FF), then optionally plane 1 (0x800..0xFFF) and plane 2
// (0x1000..0x17FF), so an image is 1 to 3 planes (2048, 4096, or 6144 bytes).
// They are produced offline by the firmware repo's scripts/generate_charset.py
// and kept in sync between repos.
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
