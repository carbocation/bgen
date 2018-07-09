package bgen

import (
	"encoding/binary"
	"os"

	"github.com/carbocation/pfx"
)

type BGEN struct {
	FilePath          string
	File              *os.File
	NVariants         int
	NSamples          uint32
	Compression       int
	Layout            int
	SampleIDsPresence int
	SamplesStart      int
	VariantsStart     int
}

// Open attempts to read a bgen file located at path. If successful,
// this returns a new BGEN object. Otherwise, it returns an error.
func Open(path string) (*BGEN, error) {
	var offset int64

	b := &BGEN{
		FilePath: path,
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, pfx.Err(err)
	}
	b.File = file

	// Read the first 4 bytes to get the # of samples
	buffer := make([]byte, 4)
	_, err = b.File.ReadAt(buffer, offset)
	if err != nil {
		return nil, pfx.Err(err)
	}
	b.NSamples = binary.LittleEndian.Uint32(buffer)

	return b, nil
}
