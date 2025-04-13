package main

import "fmt"

type versionType struct {
	Major, Minor, Patch int
	Prefix, Suffix      string
}

func (v *versionType) String() string {
	if v.Major == 0 && v.Minor == 0 && v.Patch == 0 && len(v.Prefix) == 0 && len(v.Suffix) == 0 {
		return "private-dev"
	}
	return fmt.Sprintf("%s%d.%d.%d%s", v.Prefix, v.Major, v.Minor, v.Patch, v.Suffix)
}

var Version versionType

//go:generate go run ../../tools/generate_version.go
