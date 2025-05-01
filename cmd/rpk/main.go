package main

import (
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		noMode()
	}

	if d, err := os.Getwd(); err != nil {
		logf("error: %v\n", err)
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
	var zstd *zstdInfo
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

		case "--zstd":
			zstd, _ = handleZstd("")

		case "-V", "--version":
			handleArg('V')

		case "-h", "--help":
			handleArg('h')

		default:
			if len(arg) == 0 {
				panic("empty argument")
			}
			if strings.HasPrefix(arg, "--zstd=") {
				if i, err := handleZstd(arg[7:]); err != nil {
					logf("zstd format error: %v", err)
					os.Exit(1)
				} else {
					zstd = i
				}
			} else if arg[0] == '-' {
				handled := 0
				for _, r := range arg[1:] {
					if handleArg(r) {
						handled++
					} else {
						logf("warning: ignoring unsupported flag '-%v'\n", string(r))
					}
				}
				if handled == 0 {
					logf("warning: ignoring unsupported flag '%s'\n", arg)
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
		logln("error: not enough arguments for provided options")
		os.Exit(1)
	}

	if list {
		handleCommand(listArchive(name, zstd, verbose))
		return
	}

	if extract {
		handleCommand(unpackArchive(name, zstd, verbose))
		return
	}

	if !create {
		noMode()
	}

	if len(files) == 0 {
		files = append(files, "*")
	}
	handleCommand(packArchive(name, files, excludes, zstd, verbose))
}
