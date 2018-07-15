package bgen

type Variant struct {
	ID            string
	RSID          string
	Chromosome    string
	Position      uint32
	NAlleles      uint16
	Alleles       []Allele
	Probabilities *Probability
}

//func NewVariantReader() // <- iterate over variants sequentially, possibly to build an index
//func ReadVariantAt() // <- randomly seek a variant
