package bgen

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/carbocation/pfx"
)

// BGENVersion is the supported version of the BGEN file format
const BGENVersion = "1.2"

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

// BGEN is the main object used for parsing BGEN files
type BGEN struct {
	FilePath         string   // TODO: Make private, expose fully resolved path by method?
	File             *os.File // TODO: Make private, expose by method (if at all)?
	NVariants        uint32   // TODO: Make private, expose by method?
	NSamples         uint32   // TODO: Make private, expose by method?
	FlagCompression  Compression
	FlagLayout       Layout
	FlagHasSampleIDs bool
	SamplesStart     uint32 // TODO: Make private, expose by method (if at all)?
	VariantsStart    uint32 // TODO: Make private, expose by method (if at all)?
}

func (b *BGEN) Close() error {
	return pfx.Err(b.File.Close())
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
	// VariantsStart here only if Layout == 1. If Layout == 2, however, the
	// first variant is instead at variant_offset + 4.
	b.VariantsStart = binary.LittleEndian.Uint32(buffer)

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
		// Note: The reference implementation seems to also permit "0000" in
		// addition to "bgen" as an allowable string:
		// https://bitbucket.org/gavinband/bgen/src/68ed4e34bac9cdda9441661e24550c6f76021804/src/bgen.cpp#lines-99
		// We do not allow that currently.
		return pfx.Err(fmt.Errorf("The BGEN header value at offset %d is expected to resolve to the Magic Number %s (%v when printed as a byte slice), but instead resolved to byte slice %v", offsetMagicNumber, MagicNumber, []byte(MagicNumber), buffer))
	}

	if err := b.parseAtOffsetWithBuffer(headerLength, buffer); err != nil {
		return pfx.Err(err)
	}
	flags := binary.LittleEndian.Uint32(buffer)
	hasSampleIDs := (flags & (1 << 31)) >> 31
	layout := (flags & (15 << 2)) >> 2
	compression := flags & 3

	// Derived results

	if hasSampleIDs == 1 {
		b.FlagHasSampleIDs = true
	}

	if layout == 1 {
		b.FlagLayout = Layout1
	} else if layout == 2 {
		b.FlagLayout = Layout2
	} else {
		return pfx.Err(fmt.Errorf("Layout 1 and 2 are supported; layout %d is not", layout))
	}

	if compression == 0 {
		b.FlagCompression = CompressionDisabled
	} else if compression == 1 {
		b.FlagCompression = CompressionZLIB
	} else if compression == 2 {
		b.FlagCompression = CompressionZStandard
	} else {
		return pfx.Err(fmt.Errorf("Compression 0, 1, and 2 are supported; compression %d is not", compression))
	}

	return nil
}

func (b *BGEN) parseAtOffsetWithBuffer(offset int64, buffer []byte) error {
	_, err := b.File.ReadAt(buffer, offset)
	if err != nil {
		return pfx.Err(err)
	}

	return nil
}
