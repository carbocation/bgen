package bgen

type Allele string

func (a Allele) String() string {
	return string(a)
}
