package bgen

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBitReader(t *testing.T) {
	var target uint64 = 3
	data := make([]byte, 8) // Big enough to hold a uint64

	binary.LittleEndian.PutUint64(data, target)

	val := 0
	br := newBitReader(bytes.NewBuffer(data))
	for i := 0; i < len(data); i++ {
		var bit uint
		truth, err := br.ReadBit()
		if err != nil {
			t.Fatal(err)
		}
		if truth {
			bit = 1
		}
		val |= 1 << bit
	}

	if target != uint64(val) {
		t.Errorf("Got %d, expected %d", val, target)
	}
}

func TestBitReadUint(t *testing.T) {
	var target uint64 = 3
	data := make([]byte, 8) // Big enough to hold a uint64

	binary.LittleEndian.PutUint64(data, target)

	var val uint64
	br := newBitReader(bytes.NewBuffer(data))

	val, err := br.ReadUint(8)
	if err != nil {
		t.Error(err)
	}

	if target != uint64(val) {
		t.Errorf("Got %d, expected %d", val, target)
	}

	br = newBitReader(bytes.NewBuffer(data))
	val, err = br.ReadUintLittleEndian(8)
	if err != nil {
		t.Error(err)
	}

	if target != uint64(val) {
		t.Errorf("Got %d, expected %d", val, target)
	}
}

func TestBitReaderLittleEndian7Bit(t *testing.T) {
	value := []byte{93}

	br := newBitReader(bytes.NewBuffer(value))
	valBig, err := br.ReadUint(7)
	if err != nil {
		t.Error(err)
	}

	br = newBitReader(bytes.NewBuffer(value))
	valLittle, err := br.ReadUintLittleEndian(7)
	if err != nil {
		t.Error(err)
	}

	if valBig != valLittle {
		t.Errorf("First 7 bits of %d yielded %d from bigendian, different from %d from littleendian", value[0], valBig, valLittle)
	}
}

func TestBitReaderLittleEndian16Bit(t *testing.T) {
	value := []byte{93, 115}

	br := newBitReader(bytes.NewBuffer(value))
	valLittle, err := br.ReadUintLittleEndian(16)
	if err != nil {
		t.Error(err)
	}

	properLittleEndian := binary.LittleEndian.Uint16(value)

	if valLittle != uint64(properLittleEndian) {
		t.Errorf("First 12 bits of %016b\n yielded %016b from bigendian,\n different from %016b from littleendian", value, properLittleEndian, valLittle)
	}
}
