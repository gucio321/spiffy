package spiffy

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gucio321/spiffy/pkg/gcb"
	"github.com/kpango/glg"
	"github.com/rustyoz/svg"
)

type Spiffy struct {
	scale     float64
	noComment bool
	svg       *svg.Svg
}

func NewSpiffy() *Spiffy {
	return &Spiffy{
		scale: 1.0,
	}
}

// TODO: fix types
func (s *Spiffy) Scale(scale float32) *Spiffy {
	s.scale = float64(scale)
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
	parsedData, parsedErr := s.svg.ParseDrawingInstructions()
	if err != nil {
		return builder, err
	}

	builder.BeginContinousLine()
reading:
	for {
		select {
		case cmd := <-parsedData:
			if cmd == nil {
				builder.EndContinousLine()
				break reading
			}

			switch cmd.Kind {
			case svg.MoveInstruction:
				builder.EndContinousLine()
				builder.Move(gcb.BetterPt[gcb.AbsolutePos](gcb.AbsolutePos(cmd.M[0]*s.scale), gcb.AbsolutePos(cmd.M[1]*s.scale)))
				builder.BeginContinousLine()
			case svg.CircleInstruction:
				glg.Warn("Circle not implemented")
			case svg.CurveInstruction:
				builder.DrawBezier(
					10,
					builder.Current(),
					gcb.BetterPt(gcb.AbsolutePos(cmd.CurvePoints.C1[0]*s.scale), gcb.AbsolutePos(cmd.CurvePoints.C1[1]*s.scale)),
					gcb.BetterPt(gcb.AbsolutePos(cmd.CurvePoints.C2[0]*s.scale), gcb.AbsolutePos(cmd.CurvePoints.C2[1]*s.scale)),
					gcb.BetterPt(gcb.AbsolutePos(cmd.CurvePoints.T[0]*s.scale), gcb.AbsolutePos(cmd.CurvePoints.T[1]*s.scale)),
				)
			case svg.LineInstruction:
				builder.DrawLine(
					builder.Current(),
					gcb.BetterPt[gcb.AbsolutePos](gcb.AbsolutePos(cmd.M[0]*s.scale), gcb.AbsolutePos(cmd.M[1]*s.scale)),
				)
			case svg.CloseInstruction:
				glg.Warn("Close not implemented")
			case svg.PaintInstruction:
				glg.Warn("Paint not implemented")
			}
		case err := <-parsedErr:
			if err != nil {
				panic(err)
				return builder, err
			}
		}

		/*
			continue // do to skip the following xd
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
		*/
	}

	// 2.0: draw circles
	/*
		for _, c := range s.Graph.Circles {
			builder.DrawCircleFilled(gcb.BetterPoint[gcb.AbsolutePos]{gcb.AbsolutePos(c.Cx * s.scale), gcb.AbsolutePos(c.Cy * s.scale)}, (c.R * s.scale))
			builder.Separator()
		}
	*/

	// 2.1: draw rects
	/*
		for _, r := range s.Graph.Rects {
			builder.DrawRectFilled(gcb.BetterPt(gcb.AbsolutePos(r.X*s.scale), gcb.AbsolutePos(r.Y*s.scale)), gcb.BetterPt(gcb.AbsolutePos(r.X*s.scale+r.W*s.scale), gcb.AbsolutePos(r.Y*s.scale+r.H*s.scale)))
		}
	*/

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
