package bgen

type Probability struct {
	NSamples            uint32
	NAlleles            uint16
	MinimumPloidy       uint8
	MaximumPloidy       uint8
	Phased              bool
	NProbabilityBits    uint8 // nbits. Must be 1-32 inclusive (there is no uint4 which would otherwise suffice)
	SampleProbabilities []*SampleProbability
}

// SampleProbability represents the variant data for one specfific individual at
// one specific locus, including information on whether this data is missing,
// what that individual's ploidy is, and then either (1) the probabilities for
// the phased haplotype or (2) the probabilies for the genotypes.
type SampleProbability struct {
	Missing       bool
	Ploidy        uint8 // Limited to 0-63
	Probabilities []float64
}
