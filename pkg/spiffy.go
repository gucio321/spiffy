package spiffy

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gucio321/spiffy/pkg/gcb"
	"github.com/kpango/glg"
)

type Spiffy struct {
	scale     float32
	noComment bool
	Defs      Defs  `xml:"defs"`
	Graph     Graph `xml:"g"`
}

type Defs struct {
	Rects   []Rect           `xml:"rect"`
	Linears []LinearGradient `xml:"linearGradient"`
}

type Rect struct {
	X  float32 `xml:"x,attr"`
	Y  float32 `xml:"y,attr"`
	W  float32 `xml:"width,attr"`
	H  float32 `xml:"height,attr"`
	Id string  `xml:"id,attr"`
}

type LinearGradient struct {
	Id    string  `xml:"id,attr"`
	X1    float32 `xml:"x1,attr"`
	Y1    float32 `xml:"y1,attr"`
	X2    float32 `xml:"x2,attr"`
	Y2    float32 `xml:"y2,attr"`
	Stops []Stop  `xml:"stop"` // this programm wil not use that but I'll keep it just in case
}

type Stop struct {
	Offset float32 `xml:"offset,attr"`
	Style  string  `xml:"style,attr"`
}

type Graph struct {
	Circles []Circle    `xml:"circle"`
	Paths   []Path      `xml:"path"`
	Rects   []GraphRect `xml:"rect"`
	Texts   []Text      `xml:"text"`
}

type Circle struct {
	Cx   float32 `xml:"cx,attr"`
	Cy   float32 `xml:"cy,attr"`
	R    float32 `xml:"r,attr"`
	Fill string  `xml:"fill,attr"`
}

type Path struct {
	Style string `xml:"style,attr"`
	D     string `xml:"d,attr"`
	ID    string `xml:"id,attr"`
}

type GraphRect struct {
	Style string  `xml:"style,attr"`
	ID    string  `xml:"id,attr"`
	X     float32 `xml:"x,attr"`
	Y     float32 `xml:"y,attr"`
	W     float32 `xml:"width,attr"`
	H     float32 `xml:"height,attr"`
}

type Text struct {
	ID    string  `xml:"id,attr"`
	Tspan []Tspan `xml:"tspan"`
}

type Tspan struct {
	X      float32 `xml:"x,attr"`
	Y      float32 `xml:"y,attr"`
	ID     string  `xml:"id,attr"`
	Tspan2 Tspan2  `xml:"tspan"`
}

type Tspan2 struct {
	Style string `xml:"style,attr"`
	Text  string `xml:",chardata"`
}

func NewSpiffy() *Spiffy {
	return &Spiffy{
		scale: 1.0,
	}
}

func (s *Spiffy) Scale(scale float32) *Spiffy {
	s.scale = scale
	return s
}

func (s *Spiffy) NoComment() *Spiffy {
	s.noComment = true
	return s
}

// GCode returns single-purpose GCode for our project.
func (s *Spiffy) GCode() (*gcb.GCodeBuilder, error) {
	var err error

	builder := gcb.NewGCodeBuilder()

	// 1.0: draw paths
	builder.Comment("Drawing PATHS from SVG")
	for lineIdx, line := range s.Graph.Paths {
		builder.Commentf("Drawing path %d", lineIdx)
		txts := strings.Split(line.D, " ")
		var currentType PathType
		start := gcb.BetterPoint[gcb.AbsolutePos]{0, 0}

		builder.BeginContinousLine()
		for i := 0; i < len(txts); {
			t := txts[i]
			glg.Debugf("Proessing %d/%d: %s", i, len(txts), t)
			pathType, ok := PathTypeEnum[t]
			if ok {
				currentType = pathType
				glg.Debugf("Setting current operation type to %s", currentType)
				i++
			}

			var points []gcb.BetterPoint[gcb.AbsolutePos]
			switch currentType {
			case PathMoveToRel, PathMoveToAbs:
				builder.EndContinousLine()
				points, err = s.readNPts(txts, &i, 1)
				if err != nil {
					return builder, err
				}

				if currentType == PathMoveToRel {
					for i, pt := range points {
						points[i] = builder.RelToAbs(gcb.Redefine[gcb.RelativePos](pt))
					}
				}

				glg.Debugf("Moving to %v", points[0])
				start = points[0]
				builder.Move(points[0])
				builder.BeginContinousLine()
			case PathLineToAbs, PathLineToRel:
				points, err = s.readNPts(txts, &i, 1)
				if err != nil {
					return builder, err
				}

				if currentType == PathLineToRel {
					for i, pt := range points {
						points[i] = builder.RelToAbs(gcb.Redefine[gcb.RelativePos](pt))
					}
				}

				glg.Debugf("Drawing line to %v", points[0])
				builder.DrawLine(builder.Current(), points[0])
			case PathCubicBezierCurveRel, PathCubicBezierCurveAbs:
				points, err = s.readNPts(txts, &i, 3)
				if err != nil {
					return builder, err
				}

				if currentType == PathCubicBezierCurveRel {
					for i, pt := range points {
						points[i] = builder.RelToAbs(gcb.Redefine[gcb.RelativePos](pt))
					}
				}

				glg.Debugf("Drawing cubic bezier curve from: c1: %v c2: %v end: %v", points[0], points[1], points[2])

				if err := builder.DrawBezier(20, builder.Current(), points[0], points[1], points[2]); err != nil {
					return builder, err
				}

			case PathCloseAbs, PathCloseRel:
				glg.Debugf("Close path to %v", start)
				builder.DrawLine(builder.Current(), start)
			default:
				return builder, fmt.Errorf("%s: %w", currentType, errors.New("Not Implemented"))
			}
		}

		builder.EndContinousLine()

		builder.Separator()
		glg.Debugf("Path %d done. Resetting to 0,0.", lineIdx)
		builder.Move(gcb.BetterPt[gcb.AbsolutePos](0, 0))
	}

	// 2.0: draw circles
	for _, c := range s.Graph.Circles {
		builder.DrawCircleFilled(gcb.BetterPoint[gcb.AbsolutePos]{gcb.AbsolutePos(c.Cx * s.scale), gcb.AbsolutePos(c.Cy * s.scale)}, (c.R * s.scale))
		builder.Separator()
	}

	// 2.1: draw rects
	for _, r := range s.Graph.Rects {
		builder.DrawRectFilled(gcb.BetterPt(gcb.AbsolutePos(r.X*s.scale), gcb.AbsolutePos(r.Y*s.scale)), gcb.BetterPt(gcb.AbsolutePos(r.X*s.scale+r.W*s.scale), gcb.AbsolutePos(r.Y*s.scale+r.H*s.scale)))
	}

	return builder, nil
}

func ptFromStr[T ~float32](xStr, yStr string) (gcb.BetterPoint[T], error) {
	result := gcb.BetterPoint[T]{}
	x, err := strconv.ParseFloat(xStr, 32)
	if err != nil {
		return result, err
	}

	y, err := strconv.ParseFloat(yStr, 32)
	if err != nil {
		return result, err
	}

	return gcb.BetterPt(T(x), T(y)), nil
}

func parseStr[T ~float32](t string) (gcb.BetterPoint[T], error) {
	parts := strings.Split(t, ",")
	if len(parts) != 2 {
		return gcb.BetterPt[T](0, 0), fmt.Errorf("Cant split %v: %w", t, errors.New("Unexpected paths.D parts; Not implemented"))
	}

	xStr, yStr := parts[0], parts[1]

	return ptFromStr[T](xStr, yStr)
}

func (s *Spiffy) readNPts(txts []string, i *int, n int) ([]gcb.BetterPoint[gcb.AbsolutePos], error) {
	cache := make([]gcb.BetterPoint[gcb.AbsolutePos], 0, n)
	for j := *i; j < *i+n; j++ {
		glg.Debugf("Reading %d/%d: %s", j-*i+1, n, txts[j])

		pSrc, err := parseStr[gcb.AbsolutePos](txts[j])
		if err != nil {
			return nil, fmt.Errorf("Error parsing %s (%d/%d): %w", txts[j], j-*i, n, err)
		}

		cache = append(cache, pSrc.Mul(gcb.AbsolutePos(s.scale)))
	}

	*i += n

	return cache, nil
}
