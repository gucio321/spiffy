package spiffy

//go:generate stringer -type=PathType -linecomment

type PathType int

const (
	// M - move to absolute pos
	PathMoveToAbs PathType = iota // M
	// m - move to relative pos (Pn = x0 + dx, y0 + dy)
	PathMoveToRel // m
	// L - line to absolute pos
	PathLineToAbs // L
	// l - line to relative pos (Pn = x0 + dx, y0 + dy)
	PathLineToRel // l
	// H - horizontal line to absolute pos
	PathLineToHorizontalAbs // H
	// h - horizontal line to relative pos (Pn = x0 + dx, y0)
	PathLineToHorizontalRel // h
	// V - vertical line to absolute pos
	PathLineToVerticalAbs // V
	// v - vertical line to relative pos (Pn = x0, y0 + dy)
	PathLineToVerticalRel // v
	// C - cubic bezier curve to absolute pos (3 args)
	PathCubicBezierCurveAbs // C
	// c - cubic bezier curve to relative pos (3 args)(controlPoint1, controlPoint2, endPoint)
	PathCubicBezierCurveRel // c
	PathCloseAbs            // Z
	PathCloseRel            // z
	END
)

var PathTypeEnum = func() map[string]PathType {
	m := make(map[string]PathType)
	for i := PathMoveToAbs; i < END; i++ {
		m[i.String()] = i
	}
	return m
}()
