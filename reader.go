package rawpack

import (
	"encoding/binary"
	"io"
	"strings"
)

type Reader struct {
	in io.Reader
}

func NewReader(in io.Reader) *Reader {
	return &Reader{
		in: in,
	}
}

func (r *Reader) read(b []byte) (int, error) {
	n, err := r.in.Read(b)
	if err == nil && n < len(b) {
		err = io.ErrUnexpectedEOF
	}
	return n, err
}

func (r *Reader) readUint64() (v uint64, err error) {
	var buf [8]byte
	_, err = r.read(buf[:])
	if err == nil {
		v = binary.LittleEndian.Uint64(buf[:])
	}
	return
}

func (r *Reader) readString() (string, error) {
	var buf [256]byte
	var sb strings.Builder
	l, err := r.readUint64()
	for err == nil && l > 0 {
		var n int
		n, err = r.read(buf[:min(uint64(len(buf)), l)])
		if err == nil {
			sb.Write(buf[:n])
			l -= uint64(n)
		}
	}
	return sb.String(), err
}

func (r *Reader) readFileInfo(f *File) error {
	name, err := r.readString()
	if err != nil {
		return err
	}
	size, err := r.readUint64()
	if err != nil {
		return err
	}
	f.Name = name
	f.Size = size
	return nil
}

func (r *Reader) ReadSignature() (Signature, error) {
	var s Signature
	_, err := r.read(s[:])
	return s, err
}

func (r *Reader) ReadFileTable() (FileTable, error) {
	l, err := r.readUint64()
	if err == nil {
		ft := make(FileTable, l)
		for i := range ft {
			if err := r.readFileInfo(&ft[i]); err != nil {
				return nil, err
			}
		}
		return ft, nil
	}
	return nil, nil
}

func (r *Reader) ReadFile(f *File) error {
	out, err := f.Write()
	if err == nil {
		defer out.Close()
		err = r.ReadFileTo(out, f.Size)
	}
	return err
}

func (r *Reader) ReadFileTo(out io.Writer, size uint64) error {
	n, err := io.CopyN(out, r.in, int64(size))
	if err == nil && n < int64(size) {
		err = io.ErrShortWrite
	}
	return err
}
