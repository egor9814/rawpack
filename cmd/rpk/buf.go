package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

func makeIOBuffer() (buf []byte, writeSpeed float64, err error) {
	impl := func(buf []byte) (writeSpeed float64, err error) {
		var f *os.File
		f, err = os.Create(fmt.Sprintf(".%d-%d.tmp", rand.Uint64(), rand.Uint64()))
		if err != nil {
			return
		}
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				logf("warning: cannot remove temporary file %q\n", name)
			}
		}(f.Name())
		defer func(f *os.File) {
			handleClosing(f, f.Name())
		}(f)

		start := time.Now()
		_, err = f.Write(buf)
		if err != nil {
			return
		}
		err = f.Sync()
		if err != nil {
			return
		}
		elapsed := time.Since(start)

		writeSpeed = 1.0 / elapsed.Seconds()

		return
	}

	initialSize := 2 << 20 // 2MB
	bestSpeed := 0.0
	bestSize := 0

	buf = make([]byte, initialSize)

	for size := initialSize; size > 128<<10; /* 128KB */ size >>= 1 {
		s, e := impl(buf[:size])
		if e != nil {
			return nil, 0, e
		}
		if s > bestSpeed {
			bestSpeed = s
			bestSize = size
		}
	}

	if len(buf) != bestSize {
		buf = make([]byte, bestSize)
	}
	writeSpeed = bestSpeed

	return
}
