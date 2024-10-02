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
	builder := gcb.NewGCodeBuilder()

	// 1.0: draw paths
	builder.Comment("Drawing PATHS from SVG")
	for lineIdx, line := range s.Graph.Paths {
		builder.Commentf("Drawing path %d", lineIdx)
		txts := strings.Split(line.D, " ")

		var cache []gcb.BetterPoint[float32]
		var currentType PathType
		isDrawing := false

		for i := 0; i < len(txts); i++ {
			t := txts[i]
			glg.Debugf("Proessing %d/%d: %s", i, len(txts), t)
			pathType, ok := PathTypeEnum[t]
			if ok {
				currentType = pathType
				glg.Debugf("Setting current operation type to %s", currentType)
			}

			// parse floats
			pSrc, err := parseStr[float32](t)
			switch currentType {
			case PathMoveToAbs:
				if err != nil {
					continue
				}

				cache = append(cache, pSrc)
				if isDrawing {
					builder.EndContinousLine()
				}

				p := gcb.Redefine[gcb.AbsolutePos](pSrc).Mul(gcb.AbsolutePos(s.scale))
				glg.Debugf("Moving to %v", p)
				builder.Move(p)

				isDrawing = true
				builder.BeginContinousLine()
				cache = nil
			case PathMoveToRel:
				if err != nil {
					continue
				}

				if isDrawing {
					builder.EndContinousLine()
				}

				p := gcb.Redefine[gcb.RelativePos](pSrc).Mul(gcb.RelativePos(s.scale))
				glg.Debugf("Moving to %v", p)
				builder.Move(builder.RelToAbs(p))

				isDrawing = true
				builder.BeginContinousLine()
				cache = nil
			case PathCubicBezierCurveRel:
				// read 3 args
				switch {
				case err != nil:
					continue
				case cache == nil:
					cache = make([]gcb.BetterPoint[float32], 0, 3)
					fallthrough
				case len(cache) < 2:
					cache = append(cache, pSrc)
				case len(cache) == 2:
					cache = append(cache, pSrc)
					// redefine points as Abs
					p0 := gcb.Redefine[gcb.AbsolutePos](cache[0]).Mul(gcb.AbsolutePos(s.scale)).Add(builder.Current())
					p1 := gcb.Redefine[gcb.AbsolutePos](cache[1]).Mul(gcb.AbsolutePos(s.scale)).Add(builder.Current())
					p2 := gcb.Redefine[gcb.AbsolutePos](cache[2]).Mul(gcb.AbsolutePos(s.scale)).Add(builder.Current())

					glg.Debugf("Drawing cubic bezier curve from: c1: %v c2: %v end: %v", p0, p1, p2)

					builder.DrawBezierCubic(builder.Current(), p0, p1, p2)
					// clean cache
					cache = nil
				}
			case PathCubicBezierCurveAbs:
				// read 3 args
				switch {
				case err != nil:
					continue
				case cache == nil:
					cache = make([]gcb.BetterPoint[float32], 0, 3)
					fallthrough
				case len(cache) < 2:
					cache = append(cache, pSrc)
				case len(cache) == 2:
					cache = append(cache, pSrc)
					// redefine points as Abs
					p0 := gcb.Redefine[gcb.AbsolutePos](cache[0]).Mul(gcb.AbsolutePos(s.scale))
					p1 := gcb.Redefine[gcb.AbsolutePos](cache[1]).Mul(gcb.AbsolutePos(s.scale))
					p2 := gcb.Redefine[gcb.AbsolutePos](cache[2]).Mul(gcb.AbsolutePos(s.scale))
					glg.Debugf("Drawing cubic bezier curve from: c1: %v c2: %v end: %v", p0, p1, p2)

					builder.DrawBezierCubic(builder.Current(), p0, p1, p2)
					// clean cache
					cache = nil
				}
			case PathCloseAbs, PathCloseRel:
				// 1st is 1st command, 2nd is what we want:
				p1, err := parseStr[gcb.AbsolutePos](txts[1])
				if err != nil {
					return nil, err
				}

				builder.DrawLine(builder.Current(), p1)
			default:
				return nil, fmt.Errorf("%s: %w", currentType, errors.New("Not Implemented"))
			}
		}

		builder.EndContinousLine()

		builder.Separator()
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
