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
}
