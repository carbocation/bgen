package bgen

import (
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

// parseVariantAtOffset does not mutate the VariantReader.
// TODO: Offer an option with a reusable buffer to reduce allocations
func (vr *VariantReader) parseVariantAtOffset(offset int64) (*Variant, int64, error) {
	v := &Variant{}
	var err error

VariantLoop:
	for {
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
			err = fmt.Errorf("Chromosome field size is %d bytes; expected 2", stringSize)
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
				// TODO: Handle the uncompressed genotype data
				_ = vr.buffer[:uncompressedDataBlockSize]

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
				// TODO: Handle the ZLIB compressed genotype data
				_ = vr.buffer[:genoBlockLength]
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
				// TODO: Handle the uncompressed genotype data
				_ = vr.buffer[:nextDataOffset]

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
				// TODO: Handle the compressed genotype data
				_ = vr.buffer[:genoBlockDataSizeToDecompress]

				offset += int64(genoBlockDataSizeToDecompress)
			}
		}

		// TODO: actually interpret the genotype data based on which layout
		// version is being used.

		break
	}
	// log.Println(vr.b.FlagLayout)
	// log.Println(vr.b.FlagCompression)
	// panic("yah")
	// log.Println("New offset:", offset)

	return v, offset, err
}

func (vr *VariantReader) readNBytesAtOffset(N int, offset int64) error {
	if vr.buffer == nil || len(vr.buffer) < N {
		vr.buffer = make([]byte, N)
	}

	_, err := vr.b.File.ReadAt(vr.buffer[:N], offset)
	return err
}
