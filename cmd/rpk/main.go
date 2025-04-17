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

func help() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf("%s: manipulate rawpack archive format\n", exe)
	fmt.Printf("usage: %s [options...] [pattern...]\n", exe)
	fmt.Println("options:")
	fmt.Println("  -l, --list                 list archive")
	fmt.Println("  -c, --create               create archive")
	fmt.Println("  -x, --extract              extract archive")
	fmt.Println("  -f, --file <name>          set output name")
	fmt.Println("  -d, --dir <dir>            change dir")
	fmt.Println("  -e, --exclude <pattern>    exclude files")
	fmt.Println("  -v, --verbose              verbose mode")
	fmt.Println("  -V, --version              show version")
	fmt.Println("  -h, --help                 show help")
	fmt.Println()
	fmt.Println("pattern: {term}+")
	fmt.Println("  term: [*?{c}]")
	fmt.Println("  *: any printable characters")
	fmt.Println("  ?: any one printable character")
	fmt.Println("  c: specified printable character <c>")
	fmt.Println()
	fmt.Println("pattern example:")
	fmt.Println("  *.go")
	fmt.Println("  file-?.txt")
	fmt.Println()
	fmt.Println("list archive example:")
	fmt.Printf("  %s -lvf test.rpk\n", exe)
	fmt.Println("    show files in archive 'test.rpk'")
	fmt.Println()
	fmt.Println("create archive example:")
	fmt.Printf("  %s -cvfe test.rpk *.txt\n", exe)
	fmt.Println("    create archive 'test.rpk', with all files in current directory, without all '.txt' files")
	fmt.Printf("  %s -cvfd test.rpk docs\n", exe)
	fmt.Println("    create archive 'test.rpk', with all files in directory 'docs'")
	fmt.Printf("  %s -cvfe test.rpk main.go *.go\n", exe)
	fmt.Println("    create archive 'test.rpk', with all '.go' files in current directory, without 'main.go' files")
	fmt.Println()
	fmt.Println("extract archive example:")
	fmt.Printf("  %s -xvf test.rpk\n", exe)
	fmt.Println("    extract files from archive 'test.rpk'")
	fmt.Printf("  %s -xvfd test.rpk tmp\n", exe)
	fmt.Println("    extract files from archive 'test.rpk' to directory 'tmp'")
	os.Exit(0)
}

func version() {
	fmt.Printf("%s %s\n", os.Args[0], Version.String())
	os.Exit(0)
}

func openFileForRead(name string) (io.Reader, io.Closer, error) {
	if name == "-" || name == "" {
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
	if name == "-" || name == "" {
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

func readArchive(name string, list, verbose bool) error {
	if verbose {
		if list {
			fmt.Fprintf(os.Stderr, "list of files %q:\n", name)
		} else {
			fmt.Fprintf(os.Stderr, "unpacking archive %q...\n", name)
		}
	}
	r, c, err := openFileForRead(name)
	if err != nil {
		return err
	}
	if c != nil {
		defer c.Close()
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
				fmt.Fprintf(os.Stderr, "%3d/%3d> %s (%d bytes)\n", i+1, len(ft), it.Name, it.Size)
			}
		} else {
			for _, it := range ft {
				fmt.Fprintln(os.Stderr, it)
			}
		}
	} else {
		if err := chdir(); err != nil {
			return err
		}
		if verbose {
			for i, it := range ft {
				fmt.Fprintf(os.Stderr, "%3d/%3d> unpacking %s...\n", i+1, len(ft), it.Name)
				if err := archive.ReadFile(&it); err != nil {
					return err
				}
			}
		} else {
			for _, it := range ft {
				fmt.Fprintln(os.Stderr, it.Name)
				if err := archive.ReadFile(&it); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func listArchive(name string, verbose bool) error {
	return readArchive(name, true, verbose)
}

func unpackArchive(name string, verbose bool) error {
	return readArchive(name, false, verbose)
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

func findFiles(includePatterns, excludePatterns []string) (f rawpack.FileTable, err error) {
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
				}
			}
		}
		return nil
	})
	return
}

func copyBuffer(dst io.Writer, src io.Reader, size uint64, buf []byte) (written uint64, err error) {
	// copy of io.copyBuffer, without WriterTo and ReaderFrom, with log
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
			fmt.Fprintf(os.Stderr, "\r%d/%d bytes", written, size)
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
	fmt.Fprintf(os.Stderr, "\r")
	return written, err
}

var ioBuffer [1 << 15] /* 32KB */ byte

func packFile(out io.Writer, f *rawpack.File) error {
	rc, err := f.Read()
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = copyBuffer(out, rc, f.Size, ioBuffer[:])
	return err
}

func packArchive(name string, files, excludes []string, verbose bool) error {
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
			if err := packFile(archive, &it); err != nil {
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

func handleCommand(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func noMode() {
	fmt.Fprintln(os.Stderr, "error: mode not specified (-c, -x or -l), or type '--help'")
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		noMode()
	}

	if d, err := os.Getwd(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	} else {
		wd = d
	}

	var create, list, extract, verbose bool
	var name string
	excludes := make([]string, 0, 2)
	files := make([]string, 0, 2)
	waiters := make([]*string, 0, 4)
	waitersReed := 0
	handleArg := func(r rune) bool {
		switch r {
		default:
			return false

		case 'l':
			list = true

		case 'c':
			create = true

		case 'x':
			extract = true

		case 'f':
			waiters = append(waiters, &name)

		case 'd':
			waiters = append(waiters, &wd)

		case 'e':
			l := len(excludes)
			excludes = append(excludes, "")
			waiters = append(waiters, &excludes[l])

		case 'v':
			verbose = true

		case 'V':
			version()

		case 'h':
			help()
		}
		return true
	}
	for i := 1; i < len(os.Args); i++ {
		switch arg := os.Args[i]; arg {
		case "-l", "--list":
			handleArg('l')

		case "-c", "--create":
			handleArg('c')

		case "-x", "--extract":
			handleArg('x')

		case "-f", "--file":
			handleArg('f')

		case "-d", "--dir":
			handleArg('d')

		case "-e", "--exclude":
			handleArg('e')

		case "-v", "--verbose":
			handleArg('v')

		case "-V", "--version":
			handleArg('V')

		case "-h", "--help":
			handleArg('h')

		default:
			if len(arg) == 0 {
				panic("empty argument")
			}
			if arg[0] == '-' {
				handled := 0
				for _, r := range arg[1:] {
					if handleArg(r) {
						handled++
					} else {
						fmt.Fprintf(os.Stderr, "warning: ignoring unsupported flag '-%v'\n", string(r))
					}
				}
				if handled == 0 {
					fmt.Fprintf(os.Stderr, "warning: ignoring unsupported flag '%s'\n", arg)
				}
			} else if waitersReed < len(waiters) {
				*waiters[waitersReed] = arg
				waitersReed++
			} else {
				files = append(files, arg)
			}
		}
	}
	if waitersReed < len(waiters) {
		fmt.Fprintln(os.Stderr, "error: not enough arguments for provided options")
		os.Exit(1)
	}

	if list {
		handleCommand(listArchive(name, verbose))
		return
	}

	if extract {
		handleCommand(unpackArchive(name, verbose))
		return
	}

	if !create {
		noMode()
	}

	if len(files) == 0 {
		files = append(files, "*")
	}
	handleCommand(packArchive(name, files, excludes, verbose))
}
