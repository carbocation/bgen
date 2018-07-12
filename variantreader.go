package bgen

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/DataDog/zstd"
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

	if b.FlagLayout != Layout1 {
		// Layout1 contains an extra 4 bytes at the start of each variant
		vr.currentOffset += 4
	}

	return vr
}

func (vr *VariantReader) Error() error {
	return vr.err
}

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
			// denotes the number of individuals the row represents.

			// log.Println("We are on layout 1")
			if err = vr.readNBytesAtOffset(4, offset); err != nil {
				break
			}
			offset += 4
			// log.Println("Individuals represented:", binary.LittleEndian.Uint32(vr.buffer[:4]))
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
			err = fmt.Errorf("Variant %d: Chromosome field size is %d bytes; expected 2", vr.VariantsSeen, stringSize)
			break
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
				if err = vr.populateProbabilitiesLayout1(v, vr.buffer[:uncompressedDataBlockSize]); err != nil {
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
				if err = vr.populateProbabilitiesLayout1(v, vr.buffer[:genoBlockLength]); err != nil {
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

		// TODO: actually interpret the genotype data based on which layout
		// version is being used.

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

// TODO:
func (vr *VariantReader) populateProbabilitiesLayout1(v *Variant, input []byte) error {
	switch vr.b.FlagCompression {
	case CompressionDisabled:

	case CompressionZLIB:
	}

	return fmt.Errorf("Compression choice %s is not compatible with Layout %s", vr.b.FlagCompression, vr.b.FlagLayout)
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

		reader, err := zlib.NewReader(bytes.NewBuffer(input))
		if err != nil {
			return pfx.Err(err)
		}
		if _, err = io.Copy(bb, reader); err != nil {
			return pfx.Err(err)
		}
		if len(bb.Bytes()) != expectedSize {
			return pfx.Err(fmt.Errorf("Expected to decompress %d bytes, got %d", expectedSize, len(bb.Bytes())))
		}
		if err = probabilitiesFromDecompressedLayout2(v, bb.Bytes()); err != nil {
			return pfx.Err(err)
		}
	case CompressionZStandard:
		output, err := zstd.Decompress(nil, input)
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
	log.Println(*v)
	prob := ProbabilityLayout2{}
	cursor := 0
	var size int

	size = 4
	prob.NIndividuals = binary.LittleEndian.Uint32(input[cursor : cursor+size])
	cursor += size

	prob.SampleProbabilities = make([]*SampleProbability, prob.NIndividuals, prob.NIndividuals)

	size = 2
	prob.NAlleles = binary.LittleEndian.Uint16(input[cursor : cursor+size])
	cursor += size

	if prob.NAlleles != v.NAlleles {
		return pfx.Err(fmt.Errorf("NAlleles from the probability data (%d) differs from that of the variant (%d)", prob.NAlleles, v.NAlleles))
	}

	size = 1
	prob.MinimumPloidy = input[cursor]
	cursor += size

	size = 1
	prob.MaximumPloidy = input[cursor]
	cursor += size

	// For each individual (NIndividuals), there is a byte of data. The most
	// significant bit represents missing (if 1) or nonmissing. The secondmost
	// significant bit seems to be unused. The 6 least significant bits
	// represent ploidy, clamped to (0-63). (NB: 64 is the capacity of a 6-bit
	// value; 2^6 [or 1<<6-1].)
	size = 1 // byte per sample
	for i := range prob.SampleProbabilities {
		sp := &SampleProbability{}

		// Most significant bit:
		sp.Missing = (input[cursor]&(1<<7) == 1)

		// 6 least significant bits:
		sp.Ploidy = input[cursor] & (1<<6 - 1)

		prob.SampleProbabilities[i] = sp

		cursor += size
	}

	size = 1
	prob.Phased = input[cursor] == 1
	if input[cursor] > 1 {
		return pfx.Err(fmt.Errorf("Byte representing phased status was %d (neither 0 nor 1) for variant %v", input[cursor], *v))
	}
	cursor += size

	size = 1
	prob.NProbabilityBits = input[cursor]
	if input[cursor] > 32 || input[cursor] < 1 {
		return pfx.Err(fmt.Errorf("Byte representing number of bits used to store probabilty was %d (must be 1-32 inclusive) for variant %v", input[cursor], *v))
	}
	cursor += size

	// From here out, we read *bits* instead of bytes
	rdr := newBitReader(bytes.NewBuffer(input[cursor:]))

	// For the actual probabilities,
	var denom = float64(uint(1)<<prob.NProbabilityBits - 1)

	var probBits uint64
	var nCombs int
	for _, sp := range prob.SampleProbabilities {
		probBits = 0

		if !prob.Phased {
			nCombs = Choose(int(prob.NAlleles)+int(sp.Ploidy)-1, int(prob.NAlleles)-1)
		}

		if sp.Missing {
			// Missing values are represented as zeroes but are *not* skipped.
			// "Probabilities for samples with missing data (as defined by the
			// missingness/ploidy byte) are written as zeroes (note this
			// represents a change from the earlier draft of this spec; see the
			// rationale below)." So, need to jump forward by this many bytes.

			if prob.Phased {
				// The i'th sample's data contains this many *bits*:
				if _, err = rdr.ReadUint(int(prob.NProbabilityBits) * int(sp.Ploidy) * (int(prob.NAlleles) - 1)); err != nil {
					return pfx.Err(err)
				}
			} else {
				// Unphased
				if _, err = rdr.ReadUint(int(prob.NProbabilityBits) * (nCombs - 1)); err != nil {
					return pfx.Err(err)
				}
			}

			continue
		}

		// Now iterating it bits, not bytes

		if prob.Phased {
			// The sample's data contains this many bytes:
			for i := 0; i < int(sp.Ploidy); i++ {
				for j := 0; j < int(prob.NAlleles)-1; j++ {

					probBits, err = rdr.ReadUint(int(prob.NProbabilityBits))
					if err != nil {
						return pfx.Err(err)
					}
					sp.Probabilities = append(sp.Probabilities, float64(probBits)/denom)
				}
			}
		} else {
			// Unphased
			for i := 0; i < nCombs-1; i++ {

				probBits, err = rdr.ReadUint(int(prob.NProbabilityBits))
				if err != nil {
					return pfx.Err(err)
				}
				sp.Probabilities = append(sp.Probabilities, float64(probBits)/denom)
			}
		}

		// prob.SampleProbabilities = append(prob.SampleProbabilities)
	}

	v.ProbabilitiesLayout2 = &prob
	log.Printf("%+v\n%+v\n", *v, v.ProbabilitiesLayout2)
	for i, v := range v.ProbabilitiesLayout2.SampleProbabilities {
		if v != nil {
			log.Printf("%+v\n", v)
		}
		if i > 5 {
			break
		}
	}

	return fmt.Errorf("Not yet implemented")
}

// type bitReader struct {
// 	bitCursor int
// 	bytes     []byte
// }

// func newBitReader(b []byte) *bitReader {
// 	return &bitReader{
// 		bytes: b,
// 	}
// }

// func (b *bitReader) ReadIntFromNBits(n int) int {
// 	if n < 1 {
// 		return 0
// 	}

// 	b.bitCursor += n

// }
