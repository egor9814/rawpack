package main

import (
	"crypto/md5"
	"io"
)

type cryptoKey struct {
	hash  [md5.Size]byte
	index int
}

func (p *cryptoKey) reset(data []byte) {
	p.index = 0
	p.hash = md5.Sum(data)
}

func (p *cryptoKey) apply(data []byte) {
	for i, it := range data {
		data[i] = it ^ p.hash[p.index]
		p.index = (p.index + 1) % len(p.hash)
	}
}

type cryptoWriter struct {
	w io.Writer
	k cryptoKey
}

func newCryptoWriter(out io.Writer, password []byte) (w *cryptoWriter) {
	w = &cryptoWriter{
		w: out,
	}
	w.k.reset(password)
	return
}

func (w *cryptoWriter) Write(data []byte) (int, error) {
	w.k.apply(data)
	return w.w.Write(data)
}

type cryptoReader struct {
	r io.Reader
	k cryptoKey
}

func newCryptoReader(in io.Reader, password []byte) (r *cryptoReader) {
	r = &cryptoReader{
		r: in,
	}
	r.k.reset(password)
	return
}

func (r *cryptoReader) Read(data []byte) (int, error) {
	n, err := r.r.Read(data)
	if n > 0 {
		r.k.apply(data[:n])
	}
	return n, err
}
