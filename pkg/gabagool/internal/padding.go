package internal

// Padding defines spacing on all four sides of an element.
type Padding struct {
	Top    int32
	Right  int32
	Bottom int32
	Left   int32
}

// UniformPadding creates a Padding with the same value on all sides.
func UniformPadding(value int32) Padding {
	return Padding{
		Top:    value,
		Right:  value,
		Bottom: value,
		Left:   value,
	}
}
