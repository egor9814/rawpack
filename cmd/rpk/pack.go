package main

import (
	"fmt"
	"io"
	"os"

	"github.com/egor9814/rawpack"
)

func packFile(out io.Writer, f *rawpack.File, buf []byte) error {
	rc, err := f.Read()
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = copyBuffer(out, rc, f.Size, buf)
	return err
}

func packArchive(name string, files, excludes []string, zstd *zstdInfo, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "creating archive %q...\n", name)
	}

	ft, err := findFiles(files, excludes)
	if err != nil {
		return err
	}

	if len(ft) == 0 {
		fmt.Fprintln(os.Stderr, "warning: files not specified, empty archive will be created")
	}

	w, c, err := openFileForWrite(name)
	if err != nil {
		return err
	}
	if c != nil {
		defer c.Close()
	}

	if err := chdir(); err != nil {
		return err
	}

	buf, writeSpeed, err := makeIoBuffer()
	if err != nil {
		return err
	}

	fileSize := uint64(0)
	for _, it := range ft {
		fileSize += it.Size
	}

	w, c, err = zstd.wrapWriter(w, c, writeSpeed, fileSize)
	if err != nil {
		return err
	}
	if c != nil {
		defer c.Close()
	}

	archive := rawpack.NewWriter(w)
	err = archive.WriteSignature(rawpack.NewSignature())
	if err == nil {
		err = archive.WriteFileTable(ft)
	}
	if err != nil {
		return err
	}

	if verbose {
		for i, it := range ft {
			fmt.Fprintf(os.Stderr, "%3d/%3d> packing %s...\n", i+1, len(ft), it.Name)
			if err := packFile(archive, &it, buf); err != nil {
				return err
			}
		}
		fmt.Fprintf(os.Stderr, "\rdone!                                            ")
	} else {
		for _, it := range ft {
			fmt.Fprintln(os.Stderr, it.Name)
			if err := archive.WriteFile(&it); err != nil {
				return err
			}
		}
	}

	return nil
}
