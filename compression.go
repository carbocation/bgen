package bgen

// Compression indicates how (and whether) the SNP block probability is compressed
type Compression uint32

const (
	CompressionDisabled Compression = iota
	CompressionZLIB
	CompressionZStandard
)
