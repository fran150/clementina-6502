// Package assets embeds static data files that ship with the emulator so the
// binary does not depend on the current working directory (which breaks under
// `go test`) or on external files being present at runtime.
package assets

import _ "embed"

// MiaKernel is the Clementina MIA bootstrap ROM image
// (assets/computer/mia/kernel.bin), embedded into the binary at build time.
//
// It is bound at build time, so editing kernel.bin is picked up on the next
// build (including `go run`, which rebuilds each invocation).
//
//go:embed computer/mia/kernel.bin
var MiaKernel []byte
