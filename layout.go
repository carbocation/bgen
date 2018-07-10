package bgen

// Layout is a versioned variant structured outlined by the BGEN spec
type Layout uint32

const (
	Layout1 Layout = iota
	Layout2
)

func (l Layout) String() string {
	switch l {
	case Layout1:
		return "Layout1"
	case Layout2:
		return "Layout2"

	default:
		return "Illegal selection"
	}
}
