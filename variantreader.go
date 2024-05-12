package bgen

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/carbocation/pfx"
)

type VariantReader struct {
	VariantsSeen  uint32
	b             *BGEN
	currentOffset uint32
	err           error

	// Cached values
	buffer []byte
}

func (b *BGEN) NewVariantReader() *VariantReader {
	vr := &VariantReader{
		currentOffset: b.VariantsStart,
		b:             b,
	}

	vr.currentOffset += 4

	return vr
}

func (vr *VariantReader) Error() error {
	return vr.err
}

// Read extracts the next variant and its genotype probabilities from the
// bitstream. If there are no variants left to read, Read returns nil. If there
// is a true error, Read populates the error value on the VariantReader, which
// can be read by calling the Error() method on the VariantReader.
func (vr *VariantReader) Read() *Variant {
	v, newOffset, err := vr.parseVariantAtOffset(int64(vr.currentOffset))
	if err != nil {
		if err == io.EOF {
			return nil
		}
		vr.err = pfx.Err(err)
	}

	vr.VariantsSeen++
	vr.currentOffset = uint32(newOffset)

	return v
}

// ReadAt extracts the variant and its genotype probabilities from the bitstream
// at the specified offset. Otherwise, it behaves like Read().
func (vr *VariantReader) ReadAt(byteOffset int64) *Variant {
	v, newOffset, err := vr.parseVariantAtOffset(byteOffset)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		vr.err = pfx.Err(err)
	}

	vr.VariantsSeen++
	vr.currentOffset = uint32(newOffset)

	return v
}

// parseVariantAtOffset makes heavy use of readNBytesAtOffset to read one
// variant starting at the given offset. readNBytesAtOffset does mutate
// *VariantReader by modifying its buffer to reduce allocations.
func (vr *VariantReader) parseVariantAtOffset(offset int64) (*Variant, int64, error) {
	v := &Variant{}
	var err error

VariantLoop:
	for {
		if vr.b.FlagLayout == Layout1 {
			// Layout1 has 4 extra bytes at the start of each variant that
			// denotes the number of individuals the row represents. In Layout2,
			// this is denoted within the genotype probability block.
			if err = vr.readNBytesAtOffset(4, offset); err != nil {
				break
			}
			offset += 4
			v.NSamples = binary.LittleEndian.Uint32(vr.buffer[:4])
		}

		// ID:
		if err = vr.readNBytesAtOffset(2, offset); err != nil {
			break
		}
		offset += 2
		stringSize := int(binary.LittleEndian.Uint16(vr.buffer[:2]))
		if err = vr.readNBytesAtOffset(stringSize, offset); err != nil {
			break
		}
		v.ID = string(vr.buffer[:stringSize])
		offset += int64(stringSize)

		// RSID
		if err = vr.readNBytesAtOffset(2, offset); err != nil {
			break
		}
		offset += 2
		stringSize = int(binary.LittleEndian.Uint16(vr.buffer[:2]))
		if err = vr.readNBytesAtOffset(stringSize, offset); err != nil {
			break
		}
		v.RSID = string(vr.buffer[:stringSize])
		offset += int64(stringSize)

		// Chrom
		if err = vr.readNBytesAtOffset(2, offset); err != nil {
			break
		}
		offset += 2
		stringSize = int(binary.LittleEndian.Uint16(vr.buffer[:2]))
		if stringSize != 2 {
			// This is OK; nothing to do. Reminder to self: We read 2 bytes to
			// discover how many characters are in the chromosome field. This
			// does *not* mean that the chromosome field must be 2 characters
			// long. The chr can be 1 character, 20 characters, whatever.
		}
		if err = vr.readNBytesAtOffset(stringSize, offset); err != nil {
			break
		}
		v.Chromosome = string(vr.buffer[:stringSize])
		offset += int64(stringSize)

		// Position
		if err = vr.readNBytesAtOffset(4, offset); err != nil {
			break
		}
		offset += 4
		v.Position = binary.LittleEndian.Uint32(vr.buffer[:4])

		// NAlleles
		if vr.b.FlagLayout == Layout1 {
			// Assumed to be 2 in Layout1
			v.NAlleles = 2
		} else if vr.b.FlagLayout == Layout2 {
			if err = vr.readNBytesAtOffset(2, offset); err != nil {
				break
			}
			offset += 2
			v.NAlleles = binary.LittleEndian.Uint16(vr.buffer[:2])
		}

		// Allele slice
		var alleleLength int
		for i := uint16(0); i < v.NAlleles; i++ {
			if err = vr.readNBytesAtOffset(4, offset); err != nil {
				break VariantLoop
			}
			offset += 4
			alleleLength = int(binary.LittleEndian.Uint32(vr.buffer[:4]))

			if err = vr.readNBytesAtOffset(alleleLength, offset); err != nil {
				break VariantLoop
			}
			offset += int64(alleleLength)
			v.Alleles = append(v.Alleles, Allele(string(vr.buffer[:alleleLength])))
		}

		// Genotype data
		if vr.b.FlagLayout == Layout1 {
			// From the spec: "If CompressedSNPBlocks=0 this field is omitted
			// and the length of the uncompressed data is C=6N."
			if comp := vr.b.FlagCompression; comp == CompressionDisabled {
				uncompressedDataBlockSize := int64(6 * vr.b.NSamples)
				if err = vr.readNBytesAtOffset(int(uncompressedDataBlockSize), offset); err != nil {
					break
				}
				offset += uncompressedDataBlockSize
				// Handle the uncompressed genotype data
				if err = vr.populateProbabilitiesLayout1(v, vr.buffer[:uncompressedDataBlockSize], int(uncompressedDataBlockSize)); err != nil {
					break
				}

			} else if comp == CompressionZLIB {
				if err = vr.readNBytesAtOffset(4, offset); err != nil {
					break
				}
				offset += 4
				genoBlockLength := binary.LittleEndian.Uint32(vr.buffer[:4])

				if err = vr.readNBytesAtOffset(int(genoBlockLength), offset); err != nil {
					break
				}
				offset += int64(genoBlockLength)
				// Handle the ZLIB compressed genotype data
				if err = vr.populateProbabilitiesLayout1(v, vr.buffer[:genoBlockLength], int(genoBlockLength)); err != nil {
					break
				}
			} else {
				err = fmt.Errorf("Compression choice %s is not compatible with Layout %s", vr.b.FlagCompression, vr.b.FlagLayout)
				break
			}

		} else if vr.b.FlagLayout == Layout2 {
			// The genotype layout data block for Layout2 is guaranteed to have
			// a 4 byte chunk that indicates how much data is left for this
			// block (skipping ahead by this much will bring you to the next
			// chunk).
			if err = vr.readNBytesAtOffset(4, offset); err != nil {
				break
			}
			offset += 4
			nextDataOffset := binary.LittleEndian.Uint32(vr.buffer[:4])

			if vr.b.FlagCompression == CompressionDisabled {
				// If compression is disabled, it will not have the second 4
				// byte chunk that indicates how large the data chunk is after
				// decompression.

				if err = vr.readNBytesAtOffset(int(nextDataOffset), offset); err != nil {
					break
				}
				// Handle the uncompressed genotype data
				if err = vr.populateProbabilitiesLayout2(v, vr.buffer[:nextDataOffset], int(nextDataOffset)); err != nil {
					break
				}

				offset += int64(nextDataOffset)

			} else {
				// If compression is enabled, there will be a second 4 byte
				// chunk that indicates how large the data chunk is after
				// decompression.

				if err = vr.readNBytesAtOffset(4, offset); err != nil {
					break
				}
				offset += 4
				decompressedDataLength := binary.LittleEndian.Uint32(vr.buffer[:4])
				// TODO: Is this a checksum or actually useful?
				_ = decompressedDataLength // ???

				// From the spec: "If CompressedSNPBlocks is nonzero, this is
				// C-4 bytes which can be uncompressed to form D bytes in the
				// format described below." For us, "C" is nextDataOffset.
				genoBlockDataSizeToDecompress := nextDataOffset - 4
				// Compressed geno data
				if err = vr.readNBytesAtOffset(int(genoBlockDataSizeToDecompress), offset); err != nil {
					break
				}
				// Handle the compressed genotype data
				if err = vr.populateProbabilitiesLayout2(v, vr.buffer[:genoBlockDataSizeToDecompress], int(decompressedDataLength)); err != nil {
					break
				}

				offset += int64(genoBlockDataSizeToDecompress)
			}
		}

		break
	}

	if err != nil {
		v = nil
	}
	return v, offset, err
}

func (vr *VariantReader) readNBytesAtOffset(N int, offset int64) error {
	if vr.buffer == nil || len(vr.buffer) < N {
		vr.buffer = make([]byte, N)
	}

	_, err := vr.b.File.ReadAt(vr.buffer[:N], offset)
	return err
}

func (vr *VariantReader) populateProbabilitiesLayout1(v *Variant, input []byte, expectedSize int) error {
	switch vr.b.FlagCompression {
	case CompressionDisabled:
		if len(input) != expectedSize {
			return pfx.Err(fmt.Errorf("Expected to read %d bytes, got %d", expectedSize, len(input)))
		}

		if err := probabilitiesFromDecompressedLayout1(v, input); err != nil {
			return pfx.Err(err)
		}
	case CompressionZLIB:
		if len(input) != expectedSize {
			return pfx.Err(fmt.Errorf("Expected to start with %d compressed bytes, got %d", expectedSize, len(input)))
		}

		bb := &bytes.Buffer{}

		reader, err := zlib.NewReader(bytes.NewReader(input))
		if err != nil {
			return pfx.Err(err)
		}
		defer reader.Close()
		if _, err = io.Copy(bb, reader); err != nil {
			return pfx.Err(err)
		}

		if err := probabilitiesFromDecompressedLayout1(v, bb.Bytes()); err != nil {
			return pfx.Err(err)
		}
	default:
		return fmt.Errorf("Compression choice %s is not compatible with Layout %s", vr.b.FlagCompression, vr.b.FlagLayout)
	}

	return nil
}

func probabilitiesFromDecompressedLayout1(v *Variant, input []byte) error {
	if len(input)%6 != 0 {
		return fmt.Errorf("Input contains %d bytes, which cannot be evenly divided into %d", len(input), 6)
	}

	v.MaximumPloidy = 2
	v.MinimumPloidy = 2
	v.NSamples = uint32(len(input) / 6)
	v.NAlleles = 2
	v.NProbabilityBits = 16
	v.Phased = false
	v.SampleProbabilities = make([]SampleProbability, len(input)/6, len(input)/6)

	offset := 0
	for i := range v.SampleProbabilities {
		v.SampleProbabilities[i].Missing = false
		v.SampleProbabilities[i].Ploidy = 2
		v.SampleProbabilities[i].Probabilities = make([]float64, 3, 3)

		for j := range v.SampleProbabilities[i].Probabilities {
			v.SampleProbabilities[i].Probabilities[j] = float64(binary.LittleEndian.Uint16(input[offset:offset+2])) / 32768.0 // (32768 == 1<<15)
			offset += 2
		}
	}

	if offset != len(input) {
		// log.Println(input[len(input)-1])
		return pfx.Err(fmt.Errorf("Read %d bytes, expected to read %d bytes", offset, len(input)))
	}

	return nil
}

// expectedSize acts as a checksum, ensuring that the decompressed size matches
// with expectations.
func (vr *VariantReader) populateProbabilitiesLayout2(v *Variant, input []byte, expectedSize int) error {
	switch vr.b.FlagCompression {
	case CompressionDisabled:
		if len(input) != expectedSize {
			return pfx.Err(fmt.Errorf("Expected to decompress %d bytes, got %d", expectedSize, len(input)))
		}
		if err := probabilitiesFromDecompressedLayout2(v, input); err != nil {
			return pfx.Err(err)
		}
	case CompressionZLIB:
		bb := &bytes.Buffer{}

		reader, err := zlib.NewReader(bytes.NewReader(input))
		if err != nil {
			return pfx.Err(fmt.Errorf("Tried reading %d compressed bytes: %s", len(input), err))
		}
		defer reader.Close()
		if nBytes, err := io.Copy(bb, reader); err != nil {
			return pfx.Err(fmt.Errorf("Tried copying %d decompressed bytes (expected %d compressed / %d decompressed): %s", nBytes, len(input), expectedSize, err))
		}
		if len(bb.Bytes()) != expectedSize {
			return pfx.Err(fmt.Errorf("Expected to decompress %d bytes, got %d", expectedSize, len(bb.Bytes())))
		}
		if err = probabilitiesFromDecompressedLayout2(v, bb.Bytes()); err != nil {
			return pfx.Err(err)
		}
	case CompressionZStandard:
		output, err := DecompressZStandard(nil, input)
		//output, err := zstd.Decompress(nil, input)
		if err != nil {
			return pfx.Err(err)
		}
		if len(output) != expectedSize {
			return pfx.Err(fmt.Errorf("Expected to decompress %d bytes, got %d", expectedSize, len(output)))
		}
		if err = probabilitiesFromDecompressedLayout2(v, output); err != nil {
			return pfx.Err(err)
		}
	default:
		return fmt.Errorf("Compression choice %s is not compatible with Layout %s", vr.b.FlagCompression, vr.b.FlagLayout)
	}

	return nil
}

func probabilitiesFromDecompressedLayout2(v *Variant, input []byte) (err error) {
	cursor := 0
	var size int

	size = 4
	v.NSamples = binary.LittleEndian.Uint32(input[cursor : cursor+size])
	cursor += size

	v.SampleProbabilities = make([]SampleProbability, v.NSamples, v.NSamples)

	size = 2
	v.NAlleles = binary.LittleEndian.Uint16(input[cursor : cursor+size])
	cursor += size

	size = 1
	v.MinimumPloidy = input[cursor]
	cursor += size

	size = 1
	v.MaximumPloidy = input[cursor]
	cursor += size

	// For each individual (NIndividuals), there is a byte of data. The most
	// significant bit represents missing (if 1) or nonmissing. The secondmost
	// significant bit seems to be unused. The 6 least significant bits
	// represent ploidy, clamped to (0-63). (NB: 64 is the capacity of a 6-bit
	// value; 2^6 [or 1<<6-1].)
	size = 1 // byte per sample
	for i := range v.SampleProbabilities {
		// Most significant bit:
		v.SampleProbabilities[i].Missing = (input[cursor]&(1<<7) == 1)

		// 6 least significant bits:
		v.SampleProbabilities[i].Ploidy = input[cursor] & (1<<6 - 1)

		cursor += size
	}

	size = 1
	v.Phased = input[cursor] == 1
	if input[cursor] > 1 {
		return pfx.Err(fmt.Errorf("Byte representing phased status was %d (neither 0 nor 1) for variant %v", input[cursor], *v))
	}
	cursor += size

	size = 1
	v.NProbabilityBits = input[cursor]
	if input[cursor] > 32 || input[cursor] < 1 {
		return pfx.Err(fmt.Errorf("Byte representing number of bits used to store probabilty was %d (must be 1-32 inclusive) for variant %v", input[cursor], *v))
	}
	cursor += size

	// From here out, we read *bits* instead of bytes
	// buf := bytes.NewBuffer(input[cursor:])
	// rdr := newBitReader(buf)

	rdr := newBitReader(input[cursor:], int(v.NProbabilityBits))

	// For the actual probabilities,
	denom := float64(uint64(1)<<uint64(v.NProbabilityBits) - 1)

	var probBits, pSum uint32
	var nCombs, which int
	maxCombs := Choose(int(v.NAlleles)+int(v.MaximumPloidy)-1, int(v.NAlleles)-1)

	var unsafeBackingSlice = make([]float64, maxCombs*len(v.SampleProbabilities), maxCombs*len(v.SampleProbabilities))

	for spi := range v.SampleProbabilities {
		probBits, pSum, nCombs, which = 0, 0, 0, 0

		if !v.Phased {
			nCombs = Choose(int(v.NAlleles)+int(v.SampleProbabilities[spi].Ploidy)-1, int(v.NAlleles)-1)
		}

		if v.SampleProbabilities[spi].Missing {
			// Missing values are represented as zeroes but are *not* skipped.
			// "Probabilities for samples with missing data (as defined by the
			// missingness/ploidy byte) are written as zeroes (note this
			// represents a change from the earlier draft of this spec; see the
			// rationale below)." So, need to jump forward by this many bytes.

			if v.Phased {
				// The i'th sample's data contains this many *bits*:
				for i := 0; i < int(v.SampleProbabilities[spi].Ploidy)*(int(v.NAlleles)-1); i++ {
					probBits = rdr.Next()
				}
			} else {
				// Unphased
				for i := 0; i < nCombs-1; i++ {
					probBits = rdr.Next()
				}
			}

			continue
		}

		// We share a backing slice to reduce allocations and use three-index
		// slicing to prevent end-user append() operations from modifying
		// unrelated probabilities.
		// TODO: should be nCombs instead of maxCombs?
		v.SampleProbabilities[spi].Probabilities = unsafeBackingSlice[spi*maxCombs : (spi+1)*maxCombs : (spi+1)*maxCombs]

		// Now iterating it bits, not bytes

		if v.Phased {
			// The sample's data contains this many bytes:
			which = 0
			for i := 0; i < int(v.SampleProbabilities[spi].Ploidy); i++ {
				for j := 0; j < int(v.NAlleles)-1; j++ {
					probBits = rdr.Next()

					pSum += probBits

					v.SampleProbabilities[spi].Probabilities[which] = float64(probBits) / denom
					which++
				}
			}
			v.SampleProbabilities[spi].Probabilities[maxCombs-1] = (denom - float64(pSum)) / denom
		} else {
			// Unphased
			which = 0
			for i := 0; i < nCombs-1; i++ {
				probBits = rdr.Next()

				pSum += probBits

				v.SampleProbabilities[spi].Probabilities[which] = float64(probBits) / denom
				which++
			}
			// Final combination is implied
			//sp.Probabilities = append(sp.Probabilities, (denom-float64(pSum))/denom)
			v.SampleProbabilities[spi].Probabilities[maxCombs-1] = (denom - float64(pSum)) / denom
		}
	}

	return nil
}
