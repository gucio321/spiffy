// Package gcb provides a highly-abstracted way to generate GCode 2D engravings.
package gcb

import (
	"fmt"
	"math"
	"runtime"

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

const DefaultPreamble = `;; BEGIN PREAMBUA
M413 S0 ; Disable power loss recovery
M107 ; Fan off
M104 S0 ; Set target temperature
G92 E0 ; Hotend reset
G90 ; Absolute positioning

G28 X Y ; Home X and Y axes

G0 X80 Y80 F5000.0 ; Move to start position

G91 ; Relative positioning

; START OF PRINT

M204 S2000 ; PRinting and travel speed in mm/s/s

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
	comments            bool
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
		comments:      true,
		currentP:      BetterPoint[HardwareAbsolutePos]{BaseX, BaseY},
		depth:         BaseDepth,
		headSize:      DefaultHeadSize,
		preamble:      DefaultPreamble,
		postamble:     DefaultPostamble,
		continousLine: false,
	}
}

func (b *GCodeBuilder) Comments(c bool) *GCodeBuilder {
	b.comments = c
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

// Up stops active drawing
func (b *GCodeBuilder) Up() error {
	if !b.isDrawing {
		return fmt.Errorf("called up but its already up: %w", ErrCantChangeDrawingState)
	}

	b.PushCommand(Command{
		LineComment: "Stop drawing",
		Code:        "G0",
		Args: []Arg{
			{
				Name:  "Z",
				Value: b.depth,
			},
		},
	})

	b.isDrawing = false

	return nil
}

// Down starts drawing
func (b *GCodeBuilder) Down() error {
	if b.isDrawing {
		return fmt.Errorf("called Down but its already down: %w", ErrCantChangeDrawingState)
	}

	b.PushCommand(Command{
		LineComment: "Start drawing",
		Code:        "G0",
		Args: []Arg{
			{
				Name:  "Z",
				Value: -b.depth,
			},
		},
	})

	b.isDrawing = true

	return nil
}

// BeginContinousLine starts drawing a continous line.
// Every draw command's starting point should be b.Current() (and this will be checked and will panic if not true).
// Then, no Up()/Down() will be called automatically.
func (b *GCodeBuilder) BeginContinousLine() error {
	if b.continousLine {
		return fmt.Errorf("Called BeginContinousLine but already drawing continous line: %w", ErrCantChangeDrawingState)
	}

	if err := b.Down(); err != nil {
		return err
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

// moveRel relative destination x, y.
// NOTE: moveRel does NOT call Up/Down. It just moves.
func (b *GCodeBuilder) moveRel(p BetterPoint[RelativePos]) *GCodeBuilder {
	b.currentP = b.currentP.Add(Redefine[HardwareAbsolutePos](p))

	// Push draw command
	b.PushCommand(Command{
		LineComment: fmt.Sprintf("Move to %v", b.currentP),
		Code:        "G0",
		Args: []Arg{
			{
				Name:  "X",
				Value: b.currentP.X,
			},
			{
				Name:  "Y",
				Value: b.currentP.Y,
			},
		},
	})

	validateHwAbs(b.currentP)
	return b
}

// Move moves to absolute position given
// NOTE: Move calls moveRel so does NOT call Up/Down. It just moves.
func (b *GCodeBuilder) Move(p BetterPoint[AbsolutePos]) *GCodeBuilder {
	b.Commentf("BEGIN Move(%v)", p)

	p = validateAbs(p)
	relP := b.absToRel(translate(p))
	b.moveRel(relP)

	b.Commentf("END Move(%v)", p)

	return b
}

// Comment writes comment to GCode.
func (b *GCodeBuilder) Comment(comment string) *GCodeBuilder {
	if b.comments {
		b.PushCommand(Command{
			LineComment: comment,
		})
	}

	return b
}

func (b *GCodeBuilder) Commentf(format string, args ...interface{}) *GCodeBuilder {
	return b.Comment(fmt.Sprintf(format, args...))
}

// Separator is for nice code layout
func (b *GCodeBuilder) Separator() *GCodeBuilder {
	b.Comment("")
	return b
}

// DrawLine draws a line from (x0, y0) to (x1, y1).
func (b *GCodeBuilder) DrawLine(p0, p1 BetterPoint[AbsolutePos]) *GCodeBuilder {
	b.Commentf("BEGIN DrawLine(%v, %v)", p0, p1)

	if !b.continousLine {
		// 1.1: go to x0, y0
		b.Move(p0)
		// 1.2: start drawing
		b.Down()
	} else {
		if p0 != b.Current() {
			glg.Fatalf("DrawLine called, but current position is not the same as starting point! %v != %v", p0, b.Current())
		}
	}
	// 1.3: go to x1, y1
	b.Move(p1)
	// 1.4: stop drawing
	if !b.continousLine {
		b.Up()
	}

	b.Commentf("END DrawLine(%f, %f)", p0, p1)

	return b
}

// DrawPath draws a path of lines. Closed if true, will automatically close the path by drawing line from path[n] to path[0].
func (b *GCodeBuilder) DrawLines(path ...BetterPoint[AbsolutePos]) *GCodeBuilder {
	b.Commentf("BEGIN DrawPath(%v)", path)

	if !b.continousLine {
		b.Move(path[0])
		b.Down()
	} else {
		if path[0] != b.Current() {
			glg.Fatalf("DrawPath called, but current position is not the same as starting point! %v != %v", path[0], b.Current())
		}
	}

	for i := 1; i < len(path); i++ {
		b.Commentf("Line %d", i)
		p0 := path[i]
		b.Move(p0)
	}

	if b.continousLine {
		b.Up()
	}

	b.Commentf("END DrawPath(%v)", path)

	return b
}

// DrawCircle draws circle on absolute (x,y) with radius r.
func (b *GCodeBuilder) DrawCircle(pImg BetterPoint[AbsolutePos], r float32) *GCodeBuilder {
	b.Commentf("BEGIN DrawCircle(%f, %f)", pImg, r)

	// 1.0: find x,y to move
	p := translate(pImg)
	baseP := BetterPoint[AbsolutePos]{
		X: pImg.X,
		Y: pImg.Y + AbsolutePos(r),
	}

	if !b.continousLine {
		b.Move(baseP)
		b.Down()
	} else {
		if baseP != b.Current() {
			glg.Fatalf("DrawCircle called, but current position is not the same as starting point! %v != %v", baseP, b.Current())
		}
	}

	// 1.1: do circle
	relP := b.absToRel(p)
	b.PushCommand(Command{
		LineComment: fmt.Sprintf("Draw circle with center in %v Ands at %v", relP, baseP),
		Code:        "G2",
		Args: []Arg{
			{
				Name:  "I",
				Value: relP.X,
			},
			{
				Name:  "J",
				Value: relP.Y,
			},
		},
	})

	if !b.continousLine {
		b.Up()
	}

	b.Commentf("END DrawCircle(%f, %f)", pImg, r)
	return b
}

// DrawCircleFilled draws a filled circle.
// Make sure to set headSize before.
func (b *GCodeBuilder) DrawCircleFilled(p BetterPoint[AbsolutePos], radius float32) *GCodeBuilder {
	b.Commentf("BEGIN DrawCircleFilled(%v, %f)", p, radius)

	for r := radius; r > 0; r -= float32(b.headSize) {
		b.DrawCircle(p, r)
	}

	b.Commentf("END DrawCircleFilled(%f, %f)", p, radius)

	return b
}

// DrawSector draws a sector (part of circle) on absolute (x,y) with radius r.
// start is a RADIAL angle where to start, end is a RADIAL angle where to end.
// NOTE: start/end 0 point is positive X axis. Angle is counterclockwise.
// TODO: this is not tested yet. test on e.g. .DrawSector(10,10,10,0,math.Pi/2)
func (b *GCodeBuilder) DrawSector(pImg BetterPoint[AbsolutePos], radius float32, start, end float32) *GCodeBuilder {
	b.Commentf("BEGIN DrawSector(%v, %f, %f, %f)", pImg, radius, start, end)

	// 1.0: find x,y to move
	p := translate(pImg)
	baseP := pImg.Add(BetterPoint[AbsolutePos]{
		AbsolutePos(math.Cos(float64(start)) * float64(radius)),
		AbsolutePos(math.Sin(float64(start)) * float64(radius)),
	})

	if !b.continousLine {
		b.Move(baseP)
		b.Down()
	} else {
		if baseP != b.Current() {
			glg.Fatalf("DrawSector called, but current position is not the same as starting point! %v != %v", baseP, b.Current())
		}
	}

	// 1.1: find final x,y
	finalP := pImg.Add(BetterPoint[AbsolutePos]{
		AbsolutePos(math.Cos(float64(end)) * float64(radius)),
		AbsolutePos(math.Sin(float64(end)) * float64(radius)),
	})

	hwAbsFinalP := translate(finalP)

	relFinalP := b.absToRel(translate(finalP))
	// 1.2: do circle
	relP := b.absToRel(p)

	b.PushCommand(Command{
		LineComment: fmt.Sprintf("Draw sector with center in %v Ends at %v", relP, hwAbsFinalP),
		Code:        "G2",
		Args: []Arg{
			{"I", relP.X},
			{"J", relP.Y},
			{"X", relFinalP.X},
			{"Y", relFinalP.Y},
		},
	})

	b.currentP = hwAbsFinalP

	if !b.continousLine {
		b.Up()
	}

	b.Commentf("END DrawSector(%v, %f, %f, %f)", p, radius, start, end)
	return b
}

// DrawRect draws rectangle from (x0, y0) to (x1, y1).
func (b *GCodeBuilder) DrawRect(p0, p1 BetterPoint[AbsolutePos]) *GCodeBuilder {
	b.Commentf("BEGIN DrawRect(%v, %v)", p0, p1)

	if !b.continousLine {
		b.Move(p0)
		b.Down()
	} else {
		if p0 != b.Current() {
			glg.Fatalf("DrawRect called, but current position is not the same as starting point! %v != %v", p0, b.Current())
		}
	}

	b.Move(p1)
	b.Move(p1)
	b.Move(p0)
	b.Move(p0)

	if !b.continousLine {
		b.Up()
	}

	b.Commentf("END DrawRect(%v, %v)", p0, p1)

	return b
}

// DrawRectFilled draws a filled rectangle.
func (b *GCodeBuilder) DrawRectFilled(p0, p1 BetterPoint[AbsolutePos]) *GCodeBuilder {
	b.Commentf("BEGIN DrawRectFilled(%v, %v)", p0, p1)

	delta := AbsolutePos(b.headSize)
	for p00, p11 := p0, p1; p00.X < p11.X || p00.Y < p11.Y; p00.X, p00.Y, p11.X, p11.Y = p00.X+delta, p00.Y+delta, p11.X-delta, p11.Y-delta {
		b.DrawRect(p00, p11)
	}

	b.Commentf("END DrawRectFilled(%v, %v)", p0, p1)

	return b
}

// DrawBezierCubic draws a... Bezier cubic.
// Notes for me:
/*
* G5 I  J  P  Q  X  Y:
* - I and J: relative offset from start to 1st control pt
* - P and Q: relative offset from end to 2nd control pt
* - X and Y: end point
 */
func (b *GCodeBuilder) DrawBezierCubic(start, end, control1, control2 BetterPoint[AbsolutePos]) *GCodeBuilder {
	b.Commentf("BEGIN DrawBezierCubic(%v, %v, %v, %v)", start, end, control1, control2)

	if !b.continousLine {
		// 1.0: move to start
		b.Move(start)
		// 1.1: start drawing
		b.Down()
	} else {
		if start != b.Current() {
			glg.Fatalf("DrawBezierCubic called, but current position is not the same as starting point! %v != %v", start, b.Current())
		}
	}

	// 1.2: calculate control point 1 (as relative to start)
	control1Rel := b.absToRel(translate(control1))
	// 1.3: find relative end pos
	endRel := b.absToRel(translate(end))
	// 1.4: calculate control point 2 (as relative to end)
	// according to doc it should be control2-end
	control2Rel := control2.Add(end.Mul(-1))
	// 1.5: draw
	endHwAbs := translate(end)
	b.PushCommand(Command{
		LineComment: fmt.Sprintf("Finish at %v", endHwAbs),
		Code:        "G5",
		Args: []Arg{
			{"I", control1Rel.X},
			{"J", control1Rel.Y},
			{"P", control2Rel.X},
			{"Q", control2Rel.Y},
			{"X", endRel.X},
			{"Y", endRel.Y},
		},
	})

	if !b.continousLine {
		// 1.6: stop drawing
		b.Up()
	}
	// 1.7: update current position
	b.currentP = endHwAbs

	b.Commentf("END DrawBezierCubic(%v, %v, %v, %v)", start, end, control1, control2)
	return b
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
		result += c.String(b.comments) + "\n"
	}

	result += b.postamble

	return result
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
