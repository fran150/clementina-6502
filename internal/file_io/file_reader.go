package file_io

import "os"

// FileReader interface abstracts file operations for loading binary data into memory.
// This allows for easier testing by mocking file operations.
type FileReader interface {
	Stat() (os.FileInfo, error)
	Read(p []byte) (n int, err error)
	Close() error
}
