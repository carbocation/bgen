package bgen

type VariantsIndex struct {
	FilePath    string
	Compression uint32
	Layout      uint32
	NSamples    uint32
	NVariants   uint32
	Start       uint64
}
