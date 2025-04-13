package rawpack

type FormatFlag byte

const (
	FormatNoFlags FormatFlag = 0
)

const (
	formatFlagReserved1 FormatFlag = 1 << iota
	formatFlagReserved2
	formatFlagReserved3
	formatFlagReserved4
	formatFlagReserved5

	formatFlagMask = formatFlagReserved5<<1 - 1
)

const (
	signaturePrefix = "RAW PACK FORMAT"
)

func (f FormatFlag) IsValid() bool {
	return f&formatFlagMask == f
}

func (f FormatFlag) toChar() byte {
	return byte(f + '@')
}

func formatFromChar(b byte) FormatFlag {
	return FormatFlag(b - '@')
}

type Signature [16]byte

func (s Signature) Format() FormatFlag {
	return formatFromChar(s[len(s)-1])
}

func (s *Signature) SetFormat(f FormatFlag) {
	if !f.IsValid() {
		panic("invalid format")
	}
	s[len(s)-1] = f.toChar()
}

func (s Signature) IsValid() bool {
	return string(s[:len(s)-1]) == signaturePrefix && s.Format().IsValid()
}

func NewSignature() (s Signature) {
	copy(s[:], signaturePrefix)
	s.SetFormat(FormatNoFlags)
	return
}
