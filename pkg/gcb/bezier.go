package gcb

import (
	"math"
)

func factorial(n int) int {
	if n == 0 {
		return 1
	}

	return n * factorial(n-1)
}

// refer: http://zobaczycmatematyke.krk.pl/025-Zolkos-Krakow/bezier.html
func bezier(t float32, points []BetterPoint[AbsolutePos]) BetterPoint[AbsolutePos] {
	var result BetterPoint[AbsolutePos]

	for i := 0; i < len(points); i++ {
		d := float32(factorial(len(points)-1)) /
			float32(factorial(i)*factorial(len(points)-1-i)) *
			(float32(math.Pow(float64(t), float64(i))) *
				(float32(math.Pow(float64(1-t), float64(len(points)-1-i)))))
		result.X += points[i].X * AbsolutePos(d)
		result.Y += points[i].Y * AbsolutePos(d)
	}

	return result
}
