package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/egor9814/rawpack"
)

var wd string

func isStdIOFile(name string) bool {
	return len(name) == 0 || name == "-"
}

func openFileForRead(name string) (io.Reader, io.Closer, error) {
	if isStdIOFile(name) {
		return os.Stdin, nil, nil
	} else {
		f, err := rawpack.File{Name: name}.Read()
		if err != nil {
			return nil, nil, err
		}
		return f, f, nil
	}
}

func openFileForWrite(name string) (io.Writer, io.Closer, error) {
	if isStdIOFile(name) {
		return os.Stdout, nil, nil
	} else {
		f, err := rawpack.File{Name: name}.Write()
		if err != nil {
			return nil, nil, err
		}
		return f, f, nil
	}
}

func chdir() error {
	if info, err := os.Stat(wd); err != nil {
		if err := os.MkdirAll(wd, 0755); err != nil {
			return err
		}
	} else if !info.IsDir() {
		return fmt.Errorf("expected dir at %q", wd)
	}
	return os.Chdir(wd)
}

func regexFromPattern(pattern string) (*regexp.Regexp, error) {
	pattern = filepath.ToSlash(pattern)
	var sb strings.Builder
	sb.WriteByte('^')
	for _, r := range pattern {
		switch r {
		case '*':
			sb.WriteByte('.')
			sb.WriteByte('*')
		case '?':
			sb.WriteByte('.')
		case '.', '(', ')', '+', '|', '^', '$', '[', ']', '{', '}', '\\':
			sb.WriteByte('\\')
			sb.WriteRune(r)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('$')
	return regexp.Compile(sb.String())
}

func findFiles(includePatterns, excludePatterns []string, verbose bool) (f rawpack.FileTable, err error) {
	f = make(rawpack.FileTable, 0, 32)
	err = filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		p = filepath.ToSlash(p)
		for _, it := range excludePatterns {
			r, err := regexFromPattern(it)
			if err != nil {
				return err
			}
			if r.MatchString(p) {
				return nil
			}
		}
		for _, it := range includePatterns {
			r, err := regexFromPattern(it)
			if err != nil {
				return err
			}
			if r.MatchString(p) {
				if info, err := d.Info(); err != nil {
					return err
				} else {
					f = append(f, rawpack.File{Name: p, Size: uint64(info.Size())})
					if verbose {
						logf("\r%d", len(f))
					}
				}
			}
		}
		return nil
	})
	return
}

func copyBuffer(dst io.Writer, src io.Reader, size uint64, buf []byte) (written uint64, err error) {
	// copy of io.copyBuffer, without WriterTo and ReaderFrom, with log
	src = io.LimitReader(src, int64(size))
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("invalid write result")
				}
			}
			written += uint64(nw)
			logf("\r%d/%d bytes", written, size)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

func handleCommand(err error) {
	if err != nil {
		logf("error: %v\n", err)
		os.Exit(1)
	}
}

func handleClosing(c io.Closer, tag string) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		logf("error: cannot close %q: %v\n", tag, err)
	}
}

func noMode() {
	logln("error: mode not specified (-c, -x or -l), or type '--help'")
	os.Exit(1)
}
