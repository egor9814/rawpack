package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"

	"github.com/klauspost/compress/zstd"
	"github.com/shirou/gopsutil/v3/mem"
)

type zstdInfo struct {
	memory        *uint64
	level         zstd.EncoderLevel
	threads       byte
	memoryPercent bool
	forceAuto     bool
}

func handleZstd(s string) (i *zstdInfo, err error) {
	if len(s) == 0 {
		i = &zstdInfo{
			forceAuto: true,
		}
		return
	}
	i = &zstdInfo{
		forceAuto: false,
		level:     zstd.SpeedDefault,
		threads:   1,
	}
	runes := []rune(s)
	pos := uint64(0)
	consume := func(r rune) bool {
		if pos >= uint64(len(runes)) {
			return r == 0
		}
		if runes[pos] == r {
			pos++
			return true
		}
		return false
	}
	consumeInt := func() (uint64, bool) {
		consumeDigit := func() bool {
			if pos < uint64(len(runes)) && '0' <= runes[pos] && runes[pos] <= '9' {
				pos++
				return true
			}
			return false
		}
		parseInt := func(i *uint64) bool {
			if *i == 0 {
				return false
			}
			n, err := strconv.ParseUint(string(runes[pos-*i:pos]), 10, 64)
			if err != nil {
				return false
			}
			*i = n
			return true
		}
		l := uint64(0)
		for consumeDigit() {
			l++
		}
		ok := parseInt(&l)
		return l, ok
	}
	expected := func(value, after string) error {
		return fmt.Errorf("expected %s after %s", value, after)
	}
	for pos < uint64(len(runes)) {
		if consume('a') && consume('u') && consume('t') && consume('o') {
			i.forceAuto = true
			return
		} else if consume('l') {
			if !consume('=') {
				err = expected("'='", "'l'")
				return
			}
			if consume('l') && consume('o') && consume('w') {
				i.level = zstd.SpeedFastest
			} else if consume('m') && consume('i') && consume('d') {
				i.level = zstd.SpeedDefault
			} else if consume('h') && consume('i') && consume('g') && consume('h') {
				i.level = zstd.SpeedBetterCompression
			} else {
				err = expected("'low', 'mid' or 'high'", "'l='")
				return
			}
		} else if consume('t') {
			if !consume('=') {
				err = expected("'='", "'t'")
				return
			}
			v, ok := consumeInt()
			if !ok {
				err = expected("number", "'t='")
				return
			}
			i.threads = byte(min(255, v))
		} else if consume('m') {
			if !consume('=') {
				err = expected("'='", "'m'")
				return
			}
			v, ok := consumeInt()
			if !ok {
				err = expected("number", "'m='")
				return
			}
			i.memory = new(uint64)
			*i.memory = v
			if consume('%') {
				i.memoryPercent = true
			} else if consume('G') {
				*i.memory <<= 30
			} else if consume('M') {
				*i.memory <<= 20
			} else if consume('K') {
				*i.memory <<= 10
			}
			_ = consume('B')
		} else {
			err = expected("'l', 't', 'm' or 'auto'", "'--zstd='")
			return
		}
		if !consume(',') {
			break
		}
	}
	return
}

func (i *zstdInfo) validateParameters(writeSpeed float64, size uint64, isWrite bool) error {
	freeMem, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	if i.forceAuto {
		threadsCount, err := func() (byte, error) {
			maxThreadsByMem := max(1, int(freeMem.Available/(10<<20))) // 10MB per thread

			maxThreadsByWriteSpeed := 1
			switch {
			case writeSpeed > 300:
				maxThreadsByWriteSpeed = 8
			case writeSpeed > 150:
				maxThreadsByWriteSpeed = 6
			case writeSpeed > 50:
				maxThreadsByWriteSpeed = 4
			default:
				maxThreadsByWriteSpeed = 2
			}

			return byte(min(runtime.NumCPU(), maxThreadsByMem, maxThreadsByWriteSpeed, 255)), nil
		}()
		if err != nil {
			return err
		}
		i.threads = threadsCount

		if isWrite {
			switch {
			case size < 1<<20:
				i.level = zstd.SpeedFastest
			case size < 10<<20:
				i.level = zstd.SpeedDefault
			default:
				i.level = zstd.SpeedBetterCompression
			}
		}
		i.memory = new(uint64)
		*i.memory = uint64(float64(freeMem.Available) * 0.7)
	} else {
		if i.threads == 0 {
			i.threads = byte(runtime.NumCPU())
		} else {
			i.threads = byte(min(runtime.NumCPU(), int(i.threads)))
		}
		if i.memoryPercent {
			i.memoryPercent = false
			*i.memory = uint64(float64(freeMem.Available) / 100 * max(float64(*i.memory), 100))
		}
	}
	if i.memory == nil {
		i.memory = new(uint64)
		*i.memory = 4 << 30 // 4GB
	}
	if isWrite {
		if n := *i.memory; n != 0 {
			n |= n >> 1
			n |= n >> 2
			n |= n >> 4
			n |= n >> 8
			n |= n >> 16
			n |= n >> 32
			*i.memory = n - (n >> 1)
		}
		*i.memory = min(zstd.MaxWindowSize, max(zstd.MinWindowSize, *i.memory))
	} else {
		i.threads = min(i.threads, 4)
		*i.memory = min(1<<63, max(1<<10, *i.memory))
	}
	return nil
}

func (i *zstdInfo) wrapWriter(w io.Writer, c io.Closer, writeSpeed float64, size uint64) (io.Writer, io.Closer, error) {
	if i == nil {
		return w, c, nil
	}

	if err := i.validateParameters(writeSpeed, size, true); err != nil {
		return nil, nil, err
	}

	zw, err := zstd.NewWriter(
		w,
		zstd.WithWindowSize(int(*i.memory)),
		zstd.WithEncoderLevel(i.level),
		zstd.WithEncoderConcurrency(int(i.threads)),
	)
	return zw, zw, err
}

type zstdReadWrapper struct {
	r   io.Reader
	tmp []byte
}

func (w *zstdReadWrapper) Read(b []byte) (int, error) {
	if w.tmp != nil {
		avail := len(w.tmp)
		request := len(b)
		if request <= avail {
			copy(b, w.tmp[:request])
			avail -= request
			if avail == 0 {
				w.tmp = nil
			} else {
				w.tmp = w.tmp[request:]
			}
			return request, nil
		} else {
			copy(b, w.tmp)
			n, err := w.r.Read(b[avail:])
			n += avail
			w.tmp = nil
			return n, err
		}
	}
	return w.r.Read(b)
}

func (i *zstdInfo) wrapReader(r io.Reader, writeSpeed float64) (io.Reader, error) {
	if i == nil {
		var buf = [8]byte{
			0x28, 0xb5, 0x2f, 0xfd,
		}
		n, err := r.Read(buf[4:])
		if err != nil {
			return nil, err
		}
		if n < 4 {
			return nil, errors.New("cannot detect rawpack or ZSTD signature")
		}
		r = &zstdReadWrapper{
			r:   r,
			tmp: buf[4:],
		}
		if !bytes.Equal(buf[:4], buf[4:]) {
			return r, nil
		}
		i = &zstdInfo{
			forceAuto: true,
		}
	}

	if err := i.validateParameters(writeSpeed, 0, false); err != nil {
		return nil, err
	}

	zr, err := zstd.NewReader(
		r,
		zstd.WithDecoderConcurrency(int(i.threads)),
		zstd.WithDecoderMaxMemory(*i.memory),
	)
	return zr, err
}
