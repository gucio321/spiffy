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
	repeat    struct {
		nTimes   int
		moveDown float32
	}
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

func (s *Spiffy) Repeat(nTimes int, moveDown float32) {
	s.repeat.nTimes = nTimes
	s.repeat.moveDown = moveDown
}

// GCode returns single-purpose GCode for our project.
func (s *Spiffy) GCode(builder *gcb.GCodeBuilder) error {
	var err error

	// 1.0: draw paths
	builder.Comment("Drawing PATHS from SVG")
	parsedData, parsedErr := s.svg.ParseDrawingInstructions()
	if err != nil {
		return err
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
				glg.Info("Got Move instruction")
				builder.EndContinousLine()
				builder.Move(gcb.BetterPt[gcb.AbsolutePos](gcb.AbsolutePos(cmd.M[0]*s.scale), gcb.AbsolutePos(cmd.M[1]*s.scale)))
				builder.BeginContinousLine()
			case svg.CircleInstruction:
				glg.Warn("Circle not implemented")
			case svg.CurveInstruction:
				glg.Info("Got Curve Instruction")
				builder.DrawBezier(
					10,
					builder.Current(),
					gcb.BetterPt(gcb.AbsolutePos(cmd.CurvePoints.C1[0]*s.scale), gcb.AbsolutePos(cmd.CurvePoints.C1[1]*s.scale)),
					gcb.BetterPt(gcb.AbsolutePos(cmd.CurvePoints.C2[0]*s.scale), gcb.AbsolutePos(cmd.CurvePoints.C2[1]*s.scale)),
					gcb.BetterPt(gcb.AbsolutePos(cmd.CurvePoints.T[0]*s.scale), gcb.AbsolutePos(cmd.CurvePoints.T[1]*s.scale)),
				)
			case svg.LineInstruction:
				glg.Infof("Got Line instruction")
				builder.DrawLine(
					builder.Current(),
					gcb.BetterPt[gcb.AbsolutePos](gcb.AbsolutePos(cmd.M[0]*s.scale), gcb.AbsolutePos(cmd.M[1]*s.scale)),
				)
			case svg.CloseInstruction:
				glg.Warn("Close not implemented")
			case svg.PaintInstruction:
				glg.Warn("Paint not implemented")
			default:
				glg.Errorf("Got unexpected instruction type: %v", cmd.Kind)
			}
		case err := <-parsedErr:
			if err != nil {
				panic(err)
				return err
			}
		}
	}

	// now repeat
	builder.Move(gcb.BetterPt[gcb.AbsolutePos](0, 0))
	cmds := builder.Commands()
	for i := 0; i < s.repeat.nTimes; i++ {
		builder.PushCommand(
			gcb.Command{
				LineComment: "Move down and repeate the previous sequence.",
				Code:        gcb.G0,
				Args: map[string]gcb.RelativePos{
					"Z": -1 * gcb.RelativePos(s.repeat.moveDown),
				},
			})

		builder.PushCommand(cmds...)
	}

	return nil
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
