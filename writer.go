package rawpack

import (
	"encoding/binary"
	"io"
)

type Writer struct {
	out io.Writer
}

func NewWriter(out io.Writer) *Writer {
	return &Writer{
		out: out,
	}
}

func (w *Writer) write(b []byte) error {
	n, err := w.out.Write(b)
	if err == nil && n < len(b) {
		err = io.ErrShortWrite
	}
	return err
}

func (w *Writer) writeUint64(v uint64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	return w.write(buf[:])
}

func (w *Writer) writeFileInfo(f *File) (err error) {
	err = w.writeUint64(uint64(len(f.Name)))
	if err == nil {
		err = w.write([]byte(f.Name))
	}
	if err == nil {
		err = w.writeUint64(f.Size)
	}
	return
}

func (w *Writer) WriteSignature(s Signature) error {
	return w.write(s[:])
}

func (w *Writer) WriteFileTable(ft FileTable) (err error) {
	err = w.writeUint64(uint64(len(ft)))
	if err == nil {
		for _, it := range ft {
			err = w.writeFileInfo(&it)
			if err != nil {
				break
			}
		}
	}
	return
}

func (w *Writer) WriteFile(f *File) error {
	in, err := f.Read()
	if err == nil {
		defer in.Close()
		err = w.WriteFrom(in, f.Size)
	}
	return err
}

func (w *Writer) WriteFrom(in io.Reader, size uint64) error {
	n, err := io.CopyN(w.out, in, int64(size))
	if err == nil && n < int64(size) {
		err = io.ErrShortWrite
	}
	return err
}

func (w *Writer) Write(b []byte) (int, error) {
	return w.out.Write(b)
}
