package bgen

import (
	"encoding/binary"
	"io"
)

// Via https://play.golang.org/p/rn0bAjeEGtK

type bitReader struct {
	reader io.ByteReader
	byte   byte
	offset byte
}

func newBitReader(r io.ByteReader) *bitReader {
	return &bitReader{r, 0, 0}
}

func (r *bitReader) ReadBit() (bool, error) {
	if r.offset == 8 {
		r.offset = 0
	}
	if r.offset == 0 {
		var err error
		if r.byte, err = r.reader.ReadByte(); err != nil {
			return false, err
		}
	}
	bit := (r.byte & (0x80 >> r.offset)) != 0
	r.offset++
	return bit, nil
}

func (r *bitReader) ReadUint(nbits int) (uint64, error) {
	var result uint64
	for i := nbits - 1; i >= 0; i-- {
		bit, err := r.ReadBit()
		if err != nil {
			return 0, err
		}
		if bit {
			result |= 1 << uint(i)
		}
	}
	return result, nil
}

func (r *bitReader) ReadUintLittleEndian(nbits int) (final uint64, err error) {
	// Bit order is good
	// Byte order is bad
	// Collect bytes
	// Reverse their order

	shift := 8 - nbits
	mask := 1<<uint(nbits) - 1

	loops := nbits / 8
	resid := nbits % 8
	// invMask := 8 - nbits
	// invMask := 8 % nbits

	results := make([]byte, 0, loops)

	for loop := 0; loop < loops; loop++ {
		var result byte
		for i := 8 - 1; i >= 0; i-- {
			bit, err := r.ReadBit()
			if err != nil {
				return 0, err
			}
			if bit {
				result |= 1 << uint(i)
			}
		}
		results = append(results, result)
	}
	if resid > 0 {
		var result byte
		for i := resid - 1; i >= 0; i-- {
			bit, err := r.ReadBit()
			if err != nil {
				return 0, err
			}
			if bit {
				result |= 1 << uint(i)
			}
		}
		// Mask out the max possible for this bit size and rescale up towards
		// 2^8: //result = result << uint(invMask)

		// result <<= (uint(invMask) - 1)
		// result = result<<uint(invMask) - 1
		// result = (uint(result) & uint(mask)) << uint(shift)

		result = byte((uint(result) & uint(mask)) << uint(shift))

		results = append(results, result)
	}

	if nbits > 32 {
		final = binary.LittleEndian.Uint64(results)
	} else if nbits > 16 {
		final = uint64(binary.LittleEndian.Uint32(results))
	} else if nbits > 8 {
		final = uint64(binary.LittleEndian.Uint16(results))
	} else if nbits <= 8 {
		final = uint64(results[0])
	}
	return
}
