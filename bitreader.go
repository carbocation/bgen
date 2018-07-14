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

	loops := nbits / 8
	remainder := nbits % 8
	// invMask := 8 - nbits
	// invMask := 8 % nbits

	var results []byte
	if nbits > 32 {
		results = make([]byte, 8, 8)
	} else if nbits > 16 {
		results = make([]byte, 4, 4)
	} else if nbits > 8 {
		results = make([]byte, 2, 2)
	} else {
		results = make([]byte, 1, 1)
	}

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
		results[loop] = result
	}
	if remainder > 0 {
		var result byte
		for i := remainder - 1; i >= 0; i-- {
			bit, err := r.ReadBit()
			if err != nil {
				return 0, err
			}
			if bit {
				result |= 1 << uint(i)
			}
		}

		results[loops] = result
	}

	if nbits > 32 {
		final = binary.LittleEndian.Uint64(results)
	} else if nbits > 16 {
		final = uint64(binary.LittleEndian.Uint32(results))
	} else if nbits > 8 {
		final = uint64(binary.LittleEndian.Uint16(results))
	} else {
		final = uint64(results[0])
	}
	// if nbits%8 != 0 {
	// 	return 0, pfx.Err(fmt.Errorf("Currently can only handle probabilities that are multiples of 8 bits (8, 16, 24, 32)"))
	// }
	return
}
