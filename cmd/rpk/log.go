package main

import (
	"fmt"
	"os"
)

func log(args ...any) {
	_, _ = fmt.Fprint(os.Stderr, args...)
}

func logf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}

func logln(args ...any) {
	_, _ = fmt.Fprintln(os.Stderr, args...)
}
