package main

import (
	"github.com/egor9814/rawpack"
	"io"
)

func packFile(out io.Writer, f *rawpack.File, buf []byte) error {
	rc, err := f.Read()
	if err != nil {
		return err
	}
	defer handleClosing(rc, f.Name)
	_, err = copyBuffer(out, rc, f.Size, buf)
	return err
}

func packArchive(name, password string, files, excludes []string, zstd *zstdInfo, verbose bool) error {
	if verbose {
		logln("scaning files...")
	}
	ft, err := findFiles(files, excludes, verbose)
	if err != nil {
		return err
	}
	if len(ft) == 0 {
		logln("\rwarning: files not specified, empty archive will be created")
	}

	if verbose {
		log("\rcreating archive")
		if !isStdIOFile(name) {
			logf(" %q", name)
		}
		logln("...")
	}

	w, c, err := openFileForWrite(name)
	if err != nil {
		return err
	}
	defer handleClosing(c, name)

	if err := chdir(); err != nil {
		return err
	}

	buf, writeSpeed, err := makeIOBuffer()
	if err != nil {
		return err
	}

	fileSize := uint64(len(rawpack.Signature{}))
	for _, it := range ft {
		fileSize += it.Size + uint64(len([]byte(it.Name)))
	}
	fileSize += uint64(len(ft)) * 8

	w, c, err = zstd.wrapWriter(w, c, writeSpeed, fileSize)
	if err != nil {
		return err
	}
	defer handleClosing(c, "ZSTD Compressor")

	if len(password) > 0 {
		w = newCryptoWriter(w, []byte(password))
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
			logf("\r%3d/%3d> packing %s...\n", i+1, len(ft), it.Name)
			if err := packFile(archive, &it, buf); err != nil {
				return err
			}
		}
		logln("\rdone!                                            ")
	} else {
		for _, it := range ft {
			logln(it.Name)
			if err := archive.WriteFile(&it); err != nil {
				return err
			}
		}
	}

	return nil
}
