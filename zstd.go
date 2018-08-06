package bgen

import "github.com/DataDog/zstd"

// DecompressZStandard decompresses Zstd compressed data for bgen13. As per the
// original, "Decompress src into dst. If you have a buffer to use, you can pass
// it to prevent allocation. If it is too small, or if nil is passed, a new
// buffer will be allocated and returned."
func DecompressZStandard(dst, src []byte) ([]byte, error) {
	return zstd.Decompress(dst, src)
}
