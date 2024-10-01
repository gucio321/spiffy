package spiffy

import (
	"fmt"

	"github.com/kpango/glg"
)

type (
	AbsolutePos float32
	RelativePos float32
)

const Preamble = `;; BEGIN PREAMBUA
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

const Postamble = `;; END BUA

;; BEGIN POSTABUA
M84 X Y Z E ; Disable ALL motors
;; END POSTABUA
`

const (
	BaseX, BaseY = 80, 80
	MinX, MinY   = 80, 80
	MaxX, MaxY   = 160, 160
	BaseDepth    = 20
)

type GCodeBuilder struct {
	code               string
	depth              int
	isDrawing          bool
	currentX, currentY AbsolutePos
}

func NewGCodeBuilder() *GCodeBuilder {
	return &GCodeBuilder{
		currentX: BaseX,
		currentY: BaseY,
		depth:    BaseDepth,
	}
}

func (b *GCodeBuilder) SetDepth(depth int) *GCodeBuilder {
	b.depth = depth
	return b
}

// Up stops active drawing
func (b *GCodeBuilder) Up() *GCodeBuilder {
	if !b.isDrawing {
		glg.Fatalf("Up called, but not drawing! %s", b.code)
	}

	b.code += fmt.Sprintf("G0 Z%d ; move up\n", b.depth)

	b.isDrawing = false

	return b
}

// Down starts drawing
func (b *GCodeBuilder) Down() *GCodeBuilder {
	if b.isDrawing {
		glg.Fatalf("Down called, but already drawing! %s", b.code)
	}

	b.code += fmt.Sprintf("G0 Z-%d ; move down\n", b.depth)

	b.isDrawing = true

	return b
}

// Move relatively to.
func (b *GCodeBuilder) Move(x, y RelativePos) *GCodeBuilder {
	b.code += fmt.Sprintf("G0 X%f Y%f ; move to x%[1]f y%[2]f\n", x, y)
	b.currentX += AbsolutePos(x)
	b.currentY += AbsolutePos(y)
	return b
}

// MoveAbs moves to absolute position given
func (b *GCodeBuilder) MoveAbs(x, y AbsolutePos) *GCodeBuilder {
	x, y = validateAbs(x, y)
	relX, relY := b.absToRel(x, y)
	return b.Move(relX, relY)
}

func (b *GCodeBuilder) relToAbs(x, y RelativePos) (AbsolutePos, AbsolutePos) {
	return validateAbs(b.currentX+AbsolutePos(x), b.currentY+AbsolutePos(y))
}

func (b *GCodeBuilder) absToRel(x, y AbsolutePos) (RelativePos, RelativePos) {
	return RelativePos(x - b.currentX), RelativePos(y - b.currentY)
}

func (b *GCodeBuilder) WriteComment(comment string) *GCodeBuilder {
	b.code += fmt.Sprintf("; %s\n", comment)
	return b
}

func (b *GCodeBuilder) DrawLine(x0, y0, x1, y1 AbsolutePos) *GCodeBuilder {
	x0, y0 = translate(x0, y0)
	x1, y1 = translate(x1, y1)
	// 1.1: go to x0, y0
	b.WriteComment("Draw line")
	b.MoveAbs(x0, y0)
	// 1.2: start drawing
	b.Down()
	// 1.3: go to x1, y1
	b.MoveAbs(x1, y1)
	// 1.4: stop drawing
	b.Up()
	return b
}

func (b *GCodeBuilder) DrawCircle(x, y AbsolutePos, r float32) *GCodeBuilder {
	b.WriteComment("Draw circle")
	// 1.0: find x,y to move
	x, y = translate(x, y)
	baseX := x
	baseY := y + AbsolutePos(r)
	validateAbs(baseX, baseY)
	b.MoveAbs(baseX, baseY)
	// 1.1: do circle
	relX, relY := b.absToRel(x, y)
	b.Down()
	b.code += fmt.Sprintf("G2 I%[1]f J%[2]f ; Draw circle with center in %[1]f and %[2]f with radius %[3]f\n", relX, relY, r)
	b.Up()
	return b
}

func (b *GCodeBuilder) String() string {
	return fmt.Sprintf("%s\n%s\n%s", Preamble, b.code, Postamble)
}

// GCode returns single-purpose GCode for our project.
func (s *Spiffy) GCode() (string, error) {
	builder := NewGCodeBuilder()

	builder.DrawLine(0, 0, 10, 10)
	builder.DrawCircle(10, 10, 10)

	result := builder.String()
	return result, nil
}

func validateAbs(x, y AbsolutePos) (AbsolutePos, AbsolutePos) {
	switch {
	case x < MinX:
		glg.Fatalf("Absolute position must be positive, got %f", x)
	case x > MaxX: // we assume BaseX is a center
		glg.Fatalf("Absolute position must be less than %f, got %f", BaseX*2, x)
	case y < MinY:
		glg.Fatalf("Absolute position must be positive, got %f", y)
	case y > MaxY: // we assume BaseY is a center
		glg.Fatalf("Absolute position must be less than %f, got %f", BaseY*2, y)
	}

	return x, y
}

func translate(x, y AbsolutePos) (AbsolutePos, AbsolutePos) {
	return x + MinX, y + MinY
}
