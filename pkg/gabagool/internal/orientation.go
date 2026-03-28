package internal

// DisplayOrientation specifies the clockwise rotation applied to the display output.
// When a non-Normal orientation is set, all rendering is done to an intermediate canvas
// texture which is then rotated onto the physical screen during Present().
type DisplayOrientation int

const (
	OrientationNormal    DisplayOrientation = 0   // No rotation
	OrientationRotate90  DisplayOrientation = 90  // 90° clockwise
	OrientationRotate180 DisplayOrientation = 180 // 180°
	OrientationRotate270 DisplayOrientation = 270 // 270° clockwise (90° counter-clockwise)
)
