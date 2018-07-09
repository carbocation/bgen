package bgen

import (
	"encoding/binary"
	"fmt"

	"github.com/carbocation/pfx"
)

type Sample struct {
	SampleID string
}

func ReadSamples(b *BGEN) ([]Sample, error) {
	if b.File == nil {
		return nil, pfx.Err(fmt.Errorf("b.File is nil"))
	}

	if b.FlagHasSampleIDs == 0 {
		return nil, pfx.Err(fmt.Errorf("This file indicates that it does not have sample IDs"))
	}

	samples := make([]Sample, 0, b.NSamples)

	bufferLength := make([]byte, 2)
	bufferID := make([]byte, 2)
	offset := int64(b.SamplesStart + 8) // SamplesStart is at sample_block_length, and SamplesStart+4 is at number_samples

	nSamples := int(b.NSamples)
	var sampleTextSize uint16
	for i := 0; i < nSamples; i++ {
		if err := b.parseAtOffsetWithBuffer(offset, bufferLength); err != nil {
			return nil, pfx.Err(err)
		}
		offset += 2

		sampleTextSize = binary.LittleEndian.Uint16(bufferLength)

		// resize the sample buffer to the size dictated by the result of bufferLength
		if int(sampleTextSize) > cap(bufferID) {
			bufferID = make([]byte, sampleTextSize)
		}
		bufferID = bufferID[:sampleTextSize]
		if err := b.parseAtOffsetWithBuffer(offset, bufferID); err != nil {
			return nil, pfx.Err(err)
		}

		// Copy the buffer into a string so that the buffer can be reused
		samples = append(samples, Sample{SampleID: string(bufferID)})
		offset += int64(sampleTextSize)
	}

	return samples, nil
}
