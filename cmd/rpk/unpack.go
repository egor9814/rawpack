package main

import (
	"fmt"
	"io"

	"github.com/egor9814/rawpack"
)

func unpackFile(in io.Reader, f *rawpack.File, buf []byte) error {
	wc, err := f.Write()
	if err != nil {
		return err
	}
	defer handleClosing(wc, f.Name)
	_, err = copyBuffer(wc, in, f.Size, buf)
	return err
}

func readArchive(name string, list bool, zstd *zstdInfo, verbose bool) error {
	if verbose {
		if list {
			log("list of files")
		} else {
			log("unpacking archive")
		}
		if !isStdIOFile(name) {
			logf(" %q", name)
		}
		logln("...")
	}

	buf, writeSpeed, err := makeIOBuffer()
	if err != nil {
		return err
	}

	r, c, err := openFileForRead(name)
	if err != nil {
		return err
	}
	if c != nil {
		defer handleClosing(c, name)
	}

	r, c, err = zstd.wrapReader(r, c, writeSpeed)
	if err != nil {
		return err
	}
	if c != nil {
		defer handleClosing(c, "ZSTD Decompressor")
	}

	archive := rawpack.NewReader(r)
	s, err := archive.ReadSignature()
	if err == nil {
		if !s.IsValid() {
			err = fmt.Errorf("invalid rawpack signature: %q", string(s[:]))
		}
	}
	if err != nil {
		return err
	}

	ft, err := archive.ReadFileTable()
	if err != nil {
		return err
	}

	if list {
		if verbose {
			for i, it := range ft {
				logf("%3d/%3d> %s (%d bytes)\n", i+1, len(ft), it.Name, it.Size)
			}
		} else {
			for _, it := range ft {
				logln(it)
			}
		}
	} else {
		if err := chdir(); err != nil {
			return err
		}
		if verbose {
			for i, it := range ft {
				logf("%3d/%3d> unpacking %s...\n", i+1, len(ft), it.Name)
				if err := unpackFile(archive, &it, buf); err != nil {
					return err
				}
			}
			logf("\r                                                 ")
		} else {
			for _, it := range ft {
				logln(it.Name)
				if err := archive.ReadFile(&it); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func listArchive(name string, zstd *zstdInfo, verbose bool) error {
	return readArchive(name, true, zstd, verbose)
}

func unpackArchive(name string, zstd *zstdInfo, verbose bool) error {
	return readArchive(name, false, zstd, verbose)
}
