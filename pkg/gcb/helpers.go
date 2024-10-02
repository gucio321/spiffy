package gcb

// BetterPoint is image.Point but better
type BetterPoint[PointType ~float32] struct {
	X, Y PointType
}

func (b BetterPoint[T]) Add(other BetterPoint[T]) BetterPoint[T] {
	return BetterPoint[T]{b.X + other.X, b.Y + other.Y}
}

func (b BetterPoint[T]) Mul(scalar T) BetterPoint[T] {
	return BetterPoint[T]{b.X * scalar, b.Y * scalar}
}

func BetterPt[T ~float32](x, y T) BetterPoint[T] {
	return BetterPoint[T]{x, y}
}

func Redefine[T2, T1 ~float32](a BetterPoint[T1]) BetterPoint[T2] {
	return BetterPoint[T2]{T2(a.X), T2(a.Y)}
}
