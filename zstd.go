package bgen

import (
	"github.com/klauspost/compress/zstd"
)

// DecompressZStandard decompresses Zstd compressed data for bgen13. As per the
// original, "Decompress src into dst. If you have a buffer to use, you can pass
// it to prevent allocation. If it is too small, or if nil is passed, a new
// buffer will be allocated and returned."
func DecompressZStandard(dst, src []byte) ([]byte, error) {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}

	return dec.DecodeAll(src, dst)
}
