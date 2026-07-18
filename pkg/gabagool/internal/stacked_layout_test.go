package internal

import "testing"

func TestStackedLayout(t *testing.T) {
	tests := []struct {
		name                     string
		winH, imgH, textH, gap   int32
		wantImgY, wantTextCenter int32
	}{
		{
			name: "centered group leaves image above text",
			winH: 480, imgH: 200, textH: 40, gap: 20,
			// blockH = 260, top = (480-260)/2 = 110
			wantImgY: 110, wantTextCenter: 110 + 200 + 20 + 20, // 350
		},
		{
			name: "no text collapses to image-only centering",
			winH: 480, imgH: 200, textH: 0, gap: 20,
			// blockH = 220, top = (480-220)/2 = 130
			wantImgY: 130, wantTextCenter: 130 + 200 + 20 + 0, // 350
		},
		{
			name: "oversized group pins image to top",
			winH: 300, imgH: 400, textH: 60, gap: 20,
			// blockH = 480 > 300 -> top clamped to 0
			wantImgY: 0, wantTextCenter: 0 + 400 + 20 + 30, // 450
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imgY, textCenterY := StackedLayout(tt.winH, tt.imgH, tt.textH, tt.gap)
			if imgY != tt.wantImgY {
				t.Errorf("imgY = %d, want %d", imgY, tt.wantImgY)
			}
			if textCenterY != tt.wantTextCenter {
				t.Errorf("textCenterY = %d, want %d", textCenterY, tt.wantTextCenter)
			}
		})
	}
}

// TestStackedLayoutNoOverlap asserts the invariant that matters: the text block
// never overlaps the image, and the image top is on-screen.
func TestStackedLayoutNoOverlap(t *testing.T) {
	cases := []struct{ winH, imgH, textH, gap int32 }{
		{480, 200, 40, 20},
		{300, 250, 80, 16},
		{720, 300, 120, 24},
		{200, 300, 60, 20}, // oversized
	}
	for _, c := range cases {
		imgY, textCenterY := StackedLayout(c.winH, c.imgH, c.textH, c.gap)
		imgBottom := imgY + c.imgH
		textTop := textCenterY - c.textH/2
		if textTop < imgBottom {
			t.Errorf("text overlaps image for %+v: textTop=%d imgBottom=%d", c, textTop, imgBottom)
		}
		if imgY < 0 {
			t.Errorf("image top off-screen for %+v: imgY=%d", c, imgY)
		}
	}
}
