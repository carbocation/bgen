package bgen

type VariantGenotype struct {
	NSamples     uint32
	NAlleles     uint16
	Phased       uint8
	NBits        uint8
	PLOMiss      uint8
	NCombs       int
	MinPloidy    uint8
	MaxPloidy    uint8
	Chunk        string // TODO: *char
	CurrentChunk string // TODO: *char
	VariantIDX   int    // TODO: size_t
}
