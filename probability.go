package bgen

// SampleProbability represents the variant data for one specfific individual at
// one specific locus, including information on whether this data is missing,
// what that individual's ploidy is, and then either (1) the probabilities for
// the phased haplotype or (2) the probabilies for the genotypes.
type SampleProbability struct {
	Missing       bool
	Ploidy        uint8 // Limited to 0-63
	Probabilities []float64
}
