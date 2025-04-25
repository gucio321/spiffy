package spiffy

import (
	"errors"
	"fmt"

	"github.com/gucio321/spiffy/pkg/gcb"
	"github.com/gucio321/spiffy/pkg/workspace"
	"github.com/kpango/glg"
	"github.com/rustyoz/svg"
)

type Spiffy struct {
	scale     float64
	noComment bool
	svg       *svg.Svg
	repeat    struct {
		nTimes   int
		moveDown float64
	}
	depth struct {
		workingDepth float64
		calibration  float64
	}
	workspace     *workspace.Workspace
	workspaceName string
}

func NewSpiffy() *Spiffy {
	return &Spiffy{
		workspaceName: gcb.DefaultWorkspace,
		scale:         1.0,
	}
}

func (s *Spiffy) Workspace(workspace *workspace.Workspace) {
	s.workspace = workspace
}

func (s *Spiffy) WorkspaceName(name string) {
	s.workspaceName = name
}

// Depth sets depths stuff
// workindDepth is how much will it go down to draw/stop drawing
// calibration is how much it will go down befor all
func (s *Spiffy) Depths(workingDepth, calibration float64) {
	s.depth.workingDepth = workingDepth
	s.depth.calibration = calibration
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

func (s *Spiffy) Repeat(nTimes int, moveDown float64) {
	s.repeat.nTimes = nTimes
	s.repeat.moveDown = moveDown
}

// GCode returns single-purpose GCode for our project.
func (s *Spiffy) GCode() (*gcb.GCodeBuilder, error) {
	if s.workspace == nil {
		var err error
		s.workspace, err = workspace.Get(s.workspaceName)
		if err != nil {
			return nil, fmt.Errorf("unable to get workspace from name %s: %w", s.workspaceName, err)
		}
	}

	builder := gcb.NewGCodeBuilder(s.workspace)
	if s.depth.workingDepth != 0 {
		builder.SetDepth(gcb.RelativePos(s.depth.workingDepth))
	}

	// 1.0: draw paths
	builder.Comment("Drawing PATHS from SVG")
	parsedData, parsedErr := s.svg.ParseDrawingInstructions()
	if parsedData == nil || parsedErr == nil {
		return builder, errors.New("nil parsedData or parsedErr")
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
	}

	// now repeat
	builder.Move(gcb.BetterPt[gcb.AbsolutePos](gcb.AbsolutePos(gcb.BaseX)-gcb.AbsolutePos(s.workspace.MinX), gcb.AbsolutePos(gcb.BaseY)-gcb.AbsolutePos(s.workspace.MinY)))
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

	newBuilder := gcb.NewGCodeBuilder(s.workspace)
	if s.depth.calibration != 0 {
		newBuilder.PushCommand(
			gcb.Command{
				LineComment: "Calibrate the depth (move down)",
				Code:        gcb.G0,
				Args: map[string]gcb.RelativePos{
					"Z": -1 * gcb.RelativePos(s.depth.calibration),
				},
			})
	}

	newBuilder.PushCommand(builder.Commands()...)

	return newBuilder, nil
}
