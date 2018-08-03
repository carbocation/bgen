package bgen

type Variant struct {
	// Set up front
	ID         string
	RSID       string
	Chromosome string
	Position   uint32
	NSamples   uint32 // Populated only in Layout1
	NAlleles   uint16
	Alleles    []Allele

	// Conditional based on Layout
	MinimumPloidy       uint8
	MaximumPloidy       uint8
	Phased              bool
	NProbabilityBits    uint8
	SampleProbabilities []*SampleProbability
}
