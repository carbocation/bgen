package bgen

import (
	"encoding/binary"
	"os"

	"github.com/carbocation/pfx"
)

// MagicNumber contains the value required to confirm that a file is BGEN-conformant
const MagicNumber = 1852139362

// BGEN is the main object used for parsing BGEN files
type BGEN struct {
	FilePath          string
	File              *os.File
	NVariants         uint32
	NSamples          uint32
	Compression       uint32
	Layout            uint32
	SampleIDsPresence uint32
	SamplesStart      uint32
	VariantsStart     uint32
}

// Open attempts to read a bgen file located at path. If successful,
// this returns a new BGEN object. Otherwise, it returns an error.
func Open(path string) (*BGEN, error) {
	b := &BGEN{
		FilePath: path,
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, pfx.Err(err)
	}
	b.File = file

	err = populateBGENHeader(b)
	if err != nil {
		return nil, pfx.Err(err)
	}

	return b, nil
}

func populateBGENHeader(b *BGEN) error {
	var offset int64
	var err error

	// Read the first 4 bytes to get the location where samples start
	buffer := make([]byte, 4)
	_, err = b.File.ReadAt(buffer, offset)
	if err != nil {
		return pfx.Err(err)
	}
	b.VariantsStart = binary.LittleEndian.Uint32(buffer) + 4 // Why +4???

	return nil
}
