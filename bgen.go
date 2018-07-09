package bgen

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/carbocation/pfx"
)

// MagicNumber contains the value required to confirm that a file is BGEN-conformant
const MagicNumber = "bgen"

const (
	offsetVariant        = 0
	offsetHeaderLength   = 4
	offsetNumberVariants = 8
	offsetNumberSamples  = 12
	offsetMagicNumber    = 16
	offsetFreeStorage    = 20
)

// BGENVersion is the supported version of the BGEN file format
const BGENVersion = "1.2"

// BGEN is the main object used for parsing BGEN files
type BGEN struct {
	FilePath         string
	File             *os.File
	NVariants        uint32
	NSamples         uint32
	FlagCompression  uint32
	FlagLayout       uint32
	FlagHasSampleIDs uint32
	SamplesStart     uint32
	VariantsStart    uint32
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
	// var offset int64
	var headerLength int64
	buffer := make([]byte, 4)

	_ = headerLength

	if err := b.parseAtOffsetWithBuffer(offsetVariant, buffer); err != nil {
		return pfx.Err(err)
	}
	b.VariantsStart = binary.LittleEndian.Uint32(buffer) + 4 // First variant is at variant_offset + 4. Note that (b.VariantsStart = variant_offset + 4)

	if err := b.parseAtOffsetWithBuffer(offsetHeaderLength, buffer); err != nil {
		return pfx.Err(err)
	}
	headerLength = int64(binary.LittleEndian.Uint32(buffer))

	b.SamplesStart = uint32(headerLength + 4)

	if err := b.parseAtOffsetWithBuffer(offsetNumberVariants, buffer); err != nil {
		return pfx.Err(err)
	}
	b.NVariants = binary.LittleEndian.Uint32(buffer)

	if err := b.parseAtOffsetWithBuffer(offsetNumberSamples, buffer); err != nil {
		return pfx.Err(err)
	}
	b.NSamples = binary.LittleEndian.Uint32(buffer)

	if err := b.parseAtOffsetWithBuffer(offsetMagicNumber, buffer); err != nil {
		return pfx.Err(err)
	}
	if MagicNumber != string(buffer) {
		return pfx.Err(fmt.Errorf("The BGEN header value at offset %d is expected to resolve to the Magic Number %s (%v when printed as a byte slice), but instead resolved to byte slice %v", offsetMagicNumber, MagicNumber, []byte(MagicNumber), buffer))
	}

	if err := b.parseAtOffsetWithBuffer(headerLength, buffer); err != nil {
		return pfx.Err(err)
	}
	flags := binary.LittleEndian.Uint32(buffer)
	b.FlagCompression = flags & 3
	b.FlagLayout = (flags & (15 << 2)) >> 2
	b.FlagHasSampleIDs = (flags & (1 << 31)) >> 31

	return nil
}

func (b *BGEN) parseAtOffsetWithBuffer(offset int64, buffer []byte) error {
	_, err := b.File.ReadAt(buffer, offset)
	if err != nil {
		return pfx.Err(err)
	}

	return nil
}
