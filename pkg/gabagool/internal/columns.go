package internal

// FitColumns sizes table columns to the available width from their natural
// widths, never going below minWidth.
//
// Spare room is shared out in proportion to each column's width. When there
// is too little, a column narrower than an even share of what is left keeps
// its natural width, and only the columns wider than that are narrowed and
// wrap. Shrinking every column by the same factor wrapped short text such as
// a time to save a few pixels while the long columns barely gave.
func FitColumns(natural []int32, available, minWidth int32) []int32 {
	widths := make([]int32, len(natural))
	copy(widths, natural)

	var total int32
	for _, w := range natural {
		total += w
	}
	if total == 0 {
		return widths
	}

	if total <= available {
		remaining := available - total
		for i := range widths {
			widths[i] += int32(float64(remaining) * float64(natural[i]) / float64(total))
		}
		return widths
	}

	fixed := make([]bool, len(natural))
	remaining := available
	open := len(natural)
	for open > 0 {
		share := remaining / int32(open)
		settled := false
		for i, w := range natural {
			if !fixed[i] && w <= share {
				fixed[i] = true
				remaining -= w
				open--
				settled = true
			}
		}
		if !settled {
			break
		}
	}

	var openTotal int32
	for i, w := range natural {
		if !fixed[i] {
			openTotal += w
		}
	}
	for i, w := range natural {
		if fixed[i] {
			continue
		}
		widths[i] = int32(float64(remaining) * float64(w) / float64(openTotal))
		widths[i] = max(widths[i], minWidth)
	}
	return widths
}
