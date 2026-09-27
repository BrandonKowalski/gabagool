package internal

import (
	"slices"
	"testing"
)

func sum(widths []int32) int32 {
	var total int32
	for _, w := range widths {
		total += w
	}
	return total
}

// A column narrower than its share keeps its natural width. Scaling every
// column by the same factor wrapped a five character time onto two lines to
// save a few pixels, while the long columns that needed the room barely gave.
func TestFitColumns_ShortColumnsKeepTheirWidth(t *testing.T) {
	// icon, game, platform, time
	natural := []int32{40, 500, 600, 110}
	got := FitColumns(natural, 900, 26)

	if got[0] != 40 || got[3] != 110 {
		t.Errorf("widths = %v, want the icon and time columns untouched", got)
	}
	if total := sum(got); total > 900 {
		t.Errorf("widths = %v add up to %d, more than the 900 available", got, total)
	}
	if got[1] >= 500 || got[2] >= 600 {
		t.Errorf("widths = %v, want the long columns to give up the room", got)
	}
}

// When everything fits, the spare room is shared out as it always was.
func TestFitColumns_SpareRoomIsShared(t *testing.T) {
	got := FitColumns([]int32{100, 300}, 800, 26)
	if !slices.Equal(got, []int32{200, 600}) {
		t.Errorf("widths = %v, want [200 600]", got)
	}
}

// Even when nothing fits, no column goes below the minimum.
func TestFitColumns_Minimum(t *testing.T) {
	got := FitColumns([]int32{1000, 1000, 1000}, 60, 26)
	for _, w := range got {
		if w < 26 {
			t.Errorf("widths = %v, want none under 26", got)
		}
	}
}
