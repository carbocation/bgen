package bgen

type BGEN struct {
	FilePath          string
	File              string
	NVariants         int
	NSamples          int
	Compression       int
	Layout            int
	SampleIDsPresence int
	SamplesStart      int
	VariantsStart     int
}
