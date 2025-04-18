package rawpack

import (
	"bytes"
)

type FormatFlag byte

const (
	signaturePrefix = "RAW PACK FORMAT\u0000"
)

type Signature [16]byte

func (s Signature) IsValid() bool {
	return bytes.Equal(s[:], []byte(signaturePrefix))
}

func NewSignature() (s Signature) {
	copy(s[:], signaturePrefix)
	return
}
