package spiffy

import (
	"fmt"
	"image"

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
	MinX, MinY = 80, 80
	MaxX, MaxY = 160, 160
	BaseDepth  = 20
)

// GCodeBuilder allows to build GCode. It implements several drawing methods.
// Its purpose is to convert SVG image to GCode in an easy way. (see (*Spiffy).GCode).
// NOTE: even considering the comment on HardwareAbsolutePos, all external API for this object
// uses AbsolutePos - position absolute to image you want to draw (so starting from 0,0)
type GCodeBuilder struct {
	code                string
	depth               int
	isDrawing           bool
	currentX, currentY  HardwareAbsolutePos
	preamble, postamble string
}

// NewGCodeBuilder creates new GCodeBuilder with default values.
func NewGCodeBuilder() *GCodeBuilder {
	return &GCodeBuilder{
		currentX:  BaseX,
		currentY:  BaseY,
		depth:     BaseDepth,
		preamble:  DefaultPreamble,
		postamble: DefaultPostamble,
	}
}

// SetDepth sets how deep the Heas should go.
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

// Move relative destination x, y.
// NOTE: Move does NOT call Up/Down. It just moves.
func (b *GCodeBuilder) Move(x, y RelativePos) *GCodeBuilder {
	b.currentX += HardwareAbsolutePos(x)
	b.currentY += HardwareAbsolutePos(y)
	b.code += fmt.Sprintf("G0 X%f Y%f ; move to x %[3]f y %[4]f\n", x, y, b.currentX, b.currentY)
	validateHwAbs(b.currentX, b.currentY)
	return b
}

// MoveAbs moves to absolute position given
// NOTE: MoveAbs calls Move so does NOT call Up/Down. It just moves.
func (b *GCodeBuilder) MoveAbs(x, y AbsolutePos) *GCodeBuilder {
	x, y = validateAbs(x, y)
	relX, relY := b.absToRel(translate(x, y))
	return b.Move(relX, relY)
}

// WriteComment writes comment to GCode.
func (b *GCodeBuilder) WriteComment(comment string) *GCodeBuilder {
	b.code += fmt.Sprintf("; %s\n", comment)
	return b
}

// DrawLine draws a line from (x0, y0) to (x1, y1).
func (b *GCodeBuilder) DrawLine(x0, y0, x1, y1 AbsolutePos) *GCodeBuilder {
	return b.drawLine(x0, y0, x1, y1, true)
}

func (b *GCodeBuilder) drawLine(x0, y0, x1, y1 AbsolutePos, careDrawingState bool) *GCodeBuilder {
	// 1.1: go to x0, y0
	b.WriteComment("Draw line")
	b.MoveAbs(x0, y0)
	// 1.2: start drawing
	if careDrawingState {
		b.Down()
	}
	// 1.3: go to x1, y1
	b.MoveAbs(x1, y1)
	// 1.4: stop drawing
	if careDrawingState {
		b.Up()
	}
	return b
}

// DrawPath draws a path of lines. Closed if true, will automatically close the path by drawing line from path[n] to path[0].
func (b *GCodeBuilder) DrawPath(closed bool, path ...image.Point) *GCodeBuilder {
	b.WriteComment("Drawing path")
	b.MoveAbs(AbsolutePos(path[0].X), AbsolutePos(path[0].Y))
	b.Down()
	for i := 1; i < len(path); i++ {
		b.WriteComment(fmt.Sprintf("Line %d", i))
		p0 := path[i]
		b.MoveAbs(AbsolutePos(p0.X), AbsolutePos(p0.Y))
	}

	if closed {
		b.WriteComment("Close path")
		p0 := path[0]
		b.MoveAbs(AbsolutePos(p0.X), AbsolutePos(p0.Y))
	}

	b.Up()

	b.WriteComment("Path finished")

	return b
}

// DrawCircle draws circle on absolute (x,y) with radius r.
func (b *GCodeBuilder) DrawCircle(xImg, yImg AbsolutePos, r float32) *GCodeBuilder {
	b.WriteComment("Draw circle")
	// 1.0: find x,y to move
	x, y := translate(xImg, yImg)
	baseX := xImg
	baseY := yImg + AbsolutePos(r)
	b.MoveAbs(baseX, baseY)
	// 1.1: do circle
	relX, relY := b.absToRel(x, y)
	b.Down()
	b.code += fmt.Sprintf("G2 I%[1]f J%[2]f ; Draw circle with center in %[1]f and %[2]f with radius %[3]f\n", relX, relY, r)
	b.Up()
	return b
}

// String returns built GCode.
func (b *GCodeBuilder) String() string {
	return fmt.Sprintf("%s\n%s\n%s", b.preamble, b.code, b.postamble)
}

func (b *GCodeBuilder) relToAbs(x, y RelativePos) (HardwareAbsolutePos, HardwareAbsolutePos) {
	return validateHwAbs(b.currentX+HardwareAbsolutePos(x), b.currentY+HardwareAbsolutePos(y))
}

func (b *GCodeBuilder) absToRel(x, y HardwareAbsolutePos) (RelativePos, RelativePos) {
	return RelativePos(x - b.currentX), RelativePos(y - b.currentY)
}

func validateAbs(x, y AbsolutePos) (AbsolutePos, AbsolutePos) {
	switch {
	case x < 0:
		glg.Fatalf("Absolute position must be positive, got %f", x)
	case y < 0:
		glg.Fatalf("Absolute position must be positive, got %f", y)
	}

	return x, y
}

func validateHwAbs(x, y HardwareAbsolutePos) (HardwareAbsolutePos, HardwareAbsolutePos) {
	switch {
	case x < MinX:
		glg.Fatalf("Absolute position must be larger than %f, got %f", MinX, x)
	case x > MaxX:
		glg.Fatalf("Absolute position must be less than %f, got %f", MaxX, x)
	case y < MinY:
		glg.Fatalf("Absolute position must be larger than %f, got %f", MinY, y)
	case y > MaxY:
		glg.Fatalf("Absolute position must be less than %f, got %f", MaxY, y)
	}

	return x, y
}

// translate converts AbsolutePos to HardwareAbsolutePos by adding MinX/Y
func translate(x, y AbsolutePos) (HardwareAbsolutePos, HardwareAbsolutePos) {
	return validateHwAbs(HardwareAbsolutePos(x+MinX), HardwareAbsolutePos(y+MinY))
}
