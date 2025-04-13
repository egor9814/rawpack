package rawpack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type File struct {
	Name string
	Size uint64
}

type FileTable []File

func (f File) Read() (io.ReadCloser, error) {
	return os.Open(f.Name)
}

func (f File) Write() (io.WriteCloser, error) {
	dir := filepath.Dir(f.Name)
	if info, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("expected dir at %q", dir)
	}
	return os.OpenFile(f.Name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
}
