package viewer

import (
	"image/color"
	"math"
)

func GreenToRedHSV(v float64) color.RGBA {
	// Clamp v between 0 and 1
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}

	// Interpolate hue from 120 (green) to 0 (red)
	hue := (1.0 - v) * 120.0
	return HSVtoRGB(hue, 1.0, 1.0)
}

// HSVtoRGB maps h ∈ [0, 360), s, v ∈ [0,1] to an RGBA color
func HSVtoRGB(h, s, v float64) color.RGBA {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	case h < 360:
		r, g, b = c, 0, x
	default:
		r, g, b = 0, 0, 0
	}

	return color.RGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}
