package bgen

import (
	"fmt"

	"github.com/carbocation/pfx"
)

// DecompressZStandard moves the ZStandard decompression into its own file to
// facilitate separating ZStandard compatibility (which requires cgo) into its
// own branch. As per the original, "Decompress src into dst. If you have a
// buffer to use, you can pass it to prevent allocation. If it is too small, or
// if nil is passed, a new buffer will be allocated and returned."
func DecompressZStandard(dst, src []byte) ([]byte, error) {
	return nil, pfx.Err(fmt.Errorf("To use ZStandard (BGEN format 1.3), please use the bgen13 branch"))
}
