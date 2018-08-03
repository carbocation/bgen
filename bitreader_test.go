package bgen

import (
	"encoding/binary"
	"testing"
)

func TestBitReader(t *testing.T) {
	var target uint64 = 3
	data := make([]byte, 8) // Big enough to hold a uint64

	binary.LittleEndian.PutUint64(data, target)

	val := 0
	br := newBitReader(data, 7)
	for i := 0; i < len(data); i++ {
		bit := br.getBit(i)
		val |= 1 << bit
	}

	if target != uint64(val) {
		t.Errorf("Got %d, expected %d", val, target)
	}
}

func TestBitReaderLittleEndian7Bit(t *testing.T) {
	value := []byte{93}

	br := newBitReader(value, 7)
	valBig := br.Next()

	if valBig != 93 {
		t.Errorf("First 7 bits of %d yielded %d, expected %d", value[0], valBig, 93)
	}
}

func TestBitReaderLittleEndian16Bit(t *testing.T) {
	value := []byte{93, 115}

	br := newBitReader(value, 16)
	valLittle := br.Next()

	properLittleEndian := binary.LittleEndian.Uint16(value)

	if valLittle != uint32(properLittleEndian) {
		t.Errorf("First 12 bits of %016b\n yielded %016b from bigendian,\n different from %016b from littleendian", value, properLittleEndian, valLittle)
	}
}
