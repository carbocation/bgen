package bgen

// Compression indicates how (and whether) the SNP block probability is compressed
type Compression uint32

const (
	CompressionDisabled Compression = iota
	CompressionZLIB
	CompressionZStandard
)

func (c Compression) String() string {
	switch c {
	case CompressionDisabled:
		return "CompressionDisabled"
	case CompressionZLIB:
		return "CompressionZLIB"
	case CompressionZStandard:
		return "CompressionZStandard"

	default:
		return "Illegal selection"
	}
}
