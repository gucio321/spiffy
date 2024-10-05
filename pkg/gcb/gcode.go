// Package gcb provides a highly-abstracted way to generate GCode 2D engravings.
package gcb

import (
	"fmt"
	"runtime"
	"slices"
	"strings"

	"github.com/kpango/glg"
)

type (
	// RelativePos is a position relative to currentX/currentY
	RelativePos float32
	// AbsolutePos describes position absolute on the drawing.
	// starts form 0,0
	AbsolutePos float32
	// HardwareAbsolutePos describes a coordinates on Hardware.
	// This is because our printer has an "offset" from real 0.0 that should be considered (see MinX, MinY)
	// is AbsolutePos+MinX/MinY
	HardwareAbsolutePos float32
)

const DefaultPreamble = ` ; BEGIN PREAMBUA
M413 S0            ; Disable power loss recovery
M107               ; Fan off
M104 S0            ; Set target temperature
G92 E0             ; Hotend reset
G90                ; Absolute positioning
G28 X Y            ; Home X and Y axes
G0 X80 Y80 F5000.0 ; Move to start position
G91                ; Relative positioning
M204 S2000         ; PRinting and travel speed in mm/s/s
;; END PREABUA

;; BEGIN BUA
`

const DefaultPostamble = `;; END BUA

;; BEGIN POSTABUA
M84 X Y Z E ; Disable ALL motors
;; END POSTABUA
`

const (
	// BaseX, BaseY are base coordinates for the printer.
	BaseX, BaseY = 80, 80
	// MinX, MinY are minimum coordinates for the Printers Drawing Area.
	MinX, MinY      = 80, 80
	MaxX, MaxY      = 160, 160
	BaseDepth       = 20
	DefaultHeadSize = 2
)

// GCodeBuilder allows to build GCode. It implements several drawing methods.
// Its purpose is to convert SVG image to GCode in an easy way. (see (*Spiffy).GCode).
// NOTE: even considering the comment on HardwareAbsolutePos, all external API for this object
// uses AbsolutePos - position absolute to image you want to draw (so starting from 0,0)
type GCodeBuilder struct {
	lineComments        bool
	commentsAbove       bool
	commands            []Command
	depth               int
	headSize            int
	isDrawing           bool
	currentP            BetterPoint[HardwareAbsolutePos]
	preamble, postamble string
	continousLine       bool
}

// NewGCodeBuilder creates new GCodeBuilder with default values.
func NewGCodeBuilder() *GCodeBuilder {
	return &GCodeBuilder{
		lineComments:  true,
		commentsAbove: false,
		currentP:      BetterPoint[HardwareAbsolutePos]{BaseX, BaseY},
		depth:         BaseDepth,
		headSize:      DefaultHeadSize,
		preamble:      DefaultPreamble,
		postamble:     DefaultPostamble,
		continousLine: false,
	}
}

func (b *GCodeBuilder) Comments(line, above bool) *GCodeBuilder {
	b.lineComments = line
	b.commentsAbove = above
	return b
}

// SetDepth sets how deep the Heas should go.
func (b *GCodeBuilder) SetDepth(depth int) *GCodeBuilder {
	b.depth = depth
	return b
}

// SetHeadSize sets size of the head.
func (b *GCodeBuilder) SetHeadSize(size int) *GCodeBuilder {
	b.headSize = size
	return b
}

func (b *GCodeBuilder) PushCommand(c ...Command) *GCodeBuilder {
	b.commands = append(b.commands, c...)
	return b
}

func (b *GCodeBuilder) Commands() []Command {
	return b.commands
}

// Up stops active drawing
func (b *GCodeBuilder) Up() error {
	if !b.isDrawing {
		return fmt.Errorf("called up but its already up: %w", ErrCantChangeDrawingState)
	}

	b.PushCommand(Command{
		LineComment: "Stop drawing",
		Code:        GCodeMove,
		Args: Args{
			"Z": RelativePos(b.depth),
		},
	})

	b.isDrawing = false

	return nil
}

func (b *GCodeBuilder) stopDrawing() error {
	if !b.continousLine {
		if err := b.Up(); err != nil {
			return err
		}
	}

	return nil
}

// Down starts drawing
func (b *GCodeBuilder) Down() error {
	if b.isDrawing {
		return fmt.Errorf("called Down but its already down: %w", ErrCantChangeDrawingState)
	}

	b.PushCommand(Command{
		LineComment: "Start drawing",
		Code:        GCodeMove,
		Args: Args{
			"Z": RelativePos(-b.depth),
		},
	})

	b.isDrawing = true

	return nil
}

// startDrawing moves to the starting point and calls Down
func (b *GCodeBuilder) startDrawing(p BetterPoint[AbsolutePos]) error {
	// 1.0: check if we are already drawing a continous line (if so check positions and return)
	if b.continousLine {
		if p != b.Current() {
			return fmt.Errorf("should continue drawing at %v but estimated start position is %v: %w", b.currentP, p, ErrInvalidContinousLineContinuation)
		}

		if !b.isDrawing {
			if err := b.Down(); err != nil {
				return err
			}
		}

		return nil
	}

	// 1.1: go to x0, y0
	b.Move(p)
	// 1.2: start drawing
	if err := b.Down(); err != nil {
		return err
	}

	return nil
}

// BeginContinousLine starts drawing a continous line.
// Every draw command's starting point should be b.Current() (and this will be checked and will panic if not true).
// Then, no Up()/Down() will be called automatically.
// NOTE for draw.go: every drawer should use startDrawing/endDrawing because else Begin will not be handled correctly and will cause crash. This does not call Down anymore!
func (b *GCodeBuilder) BeginContinousLine() error {
	if b.continousLine {
		return fmt.Errorf("Called BeginContinousLine but already drawing continous line: %w", ErrCantChangeDrawingState)
	}

	b.continousLine = true
	return nil
}

// EndContinousLine stops drawing a continous line.
func (b *GCodeBuilder) EndContinousLine() error {
	if !b.continousLine {
		return fmt.Errorf("Called EndContinousLine but not drawing continous line: %w", ErrCantChangeDrawingState)
	}

	if err := b.Up(); err != nil {
		return err
	}

	b.continousLine = false
	return nil
}

// Current returns current position.
func (b *GCodeBuilder) Current() BetterPoint[AbsolutePos] {
	return Redefine[AbsolutePos](b.currentP.Add(BetterPt(HardwareAbsolutePos(-MinX), HardwareAbsolutePos(-MinY))))
}

// String returns built GCode.
func (b *GCodeBuilder) String() string {
	// actual build:
	result := b.preamble
	for _, c := range b.commands {
		s := c.String(b.lineComments, b.commentsAbove)
		if s == "" {
			continue
		}

		result += s + "\n"
	}

	result += b.postamble

	// now a bit tricky part.
	// if comment is a linecomment, align it with other comments
	// if not leave.
	longest := 0
	for _, line := range strings.Split(result, "\n") {
		if strings.HasPrefix(line, " ;") || strings.HasPrefix(line, ";;") {
			continue
		}

		l := strings.Split(line, ";")[0]
		if len(l) > longest {
			longest = len(l)
		}
	}

	// align comments
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, " ;") || strings.HasPrefix(line, ";;") {
			continue
		}

		parts := strings.Split(line, ";")

		if len(parts) == 1 {
			continue // no comment
		}

		line = strings.Join(parts, fmt.Sprintf("%s;", string(slices.Repeat[[]byte]([]byte(" "), longest-len(parts[0])))))
		lines[i] = line
	}

	return strings.Join(lines, "\n")
}

func (b *GCodeBuilder) RelToAbs(p BetterPoint[RelativePos]) BetterPoint[AbsolutePos] {
	result := Redefine[AbsolutePos](p)
	result.X += AbsolutePos(b.currentP.X - MinX)
	result.Y += AbsolutePos(b.currentP.Y - MinY)
	return result
}

func (b *GCodeBuilder) relToHwAbs(p BetterPoint[RelativePos]) BetterPoint[HardwareAbsolutePos] {
	return validateHwAbs(Redefine[HardwareAbsolutePos](p).Add(b.currentP))
}

func (b *GCodeBuilder) absToRel(p BetterPoint[HardwareAbsolutePos]) BetterPoint[RelativePos] {
	return Redefine[RelativePos](p.Add(b.currentP.Mul(-1)))
}

// Dump prints all current commands to stdout
func (b *GCodeBuilder) Dump() {
	glg.Infof("Dumping commands:")
	fmt.Println(glg.Yellow(fmt.Sprintf("%#v", b)))
}

func validateAbs(p BetterPoint[AbsolutePos]) BetterPoint[AbsolutePos] {
	switch {
	case p.X < 0:
		glg.Fatalf("Absolute position must be positive, got %f", p.X)
	case p.Y < 0:
		glg.Fatalf("Absolute position must be positive, got %f", p.Y)
	}

	return p
}

func validateHwAbs(p BetterPoint[HardwareAbsolutePos]) BetterPoint[HardwareAbsolutePos] {
	switch {
	case p.X < MinX:
		_, file, line, ok := runtime.Caller(2)
		glg.Infof("Called at: %s %d %v\n", file, line, ok)
		glg.Fatalf("Absolute position must be larger than %d, got %f", MinX, p.X)
	case p.X > MaxX:
		_, file, line, ok := runtime.Caller(2)
		glg.Infof("Called at: %s %d %v\n", file, line, ok)
		glg.Fatalf("Absolute position must be less than %v, got %f", MaxX, p.X)
	case p.Y < MinY:
		_, file, line, ok := runtime.Caller(2)
		glg.Infof("Called at: %s %d %v\n", file, line, ok)
		glg.Fatalf("Absolute position must be larger than %d, got %f", MinY, p.Y)
	case p.Y > MaxY:
		_, file, line, ok := runtime.Caller(2)
		glg.Infof("Called at: %s %d %v\n", file, line, ok)
		glg.Fatalf("Absolute position must be less than %v, got %f", MaxY, p.Y)
	}

	return p
}

// translate converts AbsolutePos to HardwareAbsolutePos by adding MinX/Y
func translate(p BetterPoint[AbsolutePos]) BetterPoint[HardwareAbsolutePos] {
	return validateHwAbs(Redefine[HardwareAbsolutePos](p.Add(BetterPoint[AbsolutePos]{MinX, MinY})))
}
