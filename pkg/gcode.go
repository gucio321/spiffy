package spiffy

import (
	"fmt"
	"image"
	"math"

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
	code                string
	depth               int
	headSize            int
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
		headSize:  DefaultHeadSize,
		preamble:  DefaultPreamble,
		postamble: DefaultPostamble,
	}
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

// Up stops active drawing
func (b *GCodeBuilder) Up() *GCodeBuilder {
	if !b.isDrawing {
		glg.Fatalf("Up called, but not drawing! %s", b.code)
	}

	b.code += fmt.Sprintf("G0 Z%d ; stop drawing\n", b.depth)

	b.isDrawing = false

	return b
}

// Down starts drawing
func (b *GCodeBuilder) Down() *GCodeBuilder {
	if b.isDrawing {
		glg.Fatalf("Down called, but already drawing! %s", b.code)
	}

	b.code += fmt.Sprintf("G0 Z-%d ; start drawing\n", b.depth)

	b.isDrawing = true

	return b
}

// moveRel relative destination x, y.
// NOTE: moveRel does NOT call Up/Down. It just moves.
func (b *GCodeBuilder) moveRel(x, y RelativePos) *GCodeBuilder {
	b.currentX += HardwareAbsolutePos(x)
	b.currentY += HardwareAbsolutePos(y)
	b.code += fmt.Sprintf("G0 X%f Y%f ; move to x %[3]f y %[4]f\n", x, y, b.currentX, b.currentY)
	validateHwAbs(b.currentX, b.currentY)
	return b
}

// Move moves to absolute position given
// NOTE: Move calls moveRel so does NOT call Up/Down. It just moves.
func (b *GCodeBuilder) Move(x, y AbsolutePos) *GCodeBuilder {
	b.Commentf("BEGIN Move(%f, %f)", x, y)

	x, y = validateAbs(x, y)
	relX, relY := b.absToRel(translate(x, y))
	b.moveRel(relX, relY)

	b.Commentf("END Move(%f, %f)", x, y)

	return b
}

// Comment writes comment to GCode.
func (b *GCodeBuilder) Comment(comment string) *GCodeBuilder {
	b.code += fmt.Sprintf("; %s\n", comment)
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
func (b *GCodeBuilder) DrawLine(x0, y0, x1, y1 AbsolutePos) *GCodeBuilder {
	b.Commentf("BEGIN DrawLine(%f, %f, %f, %f)", x0, y0, x1, y1)

	// 1.1: go to x0, y0
	b.Move(x0, y0)
	// 1.2: start drawing
	b.Down()
	// 1.3: go to x1, y1
	b.Move(x1, y1)
	// 1.4: stop drawing
	b.Up()

	b.Commentf("END DrawLine(%f, %f, %f, %f)", x0, y0, x1, y1)

	return b
}

// DrawPath draws a path of lines. Closed if true, will automatically close the path by drawing line from path[n] to path[0].
func (b *GCodeBuilder) DrawPath(closed bool, path ...image.Point) *GCodeBuilder {
	b.Commentf("BEGIN DrawPath(%v, %v)", closed, path)

	b.Move(AbsolutePos(path[0].X), AbsolutePos(path[0].Y))
	b.Down()
	for i := 1; i < len(path); i++ {
		b.Commentf("Line %d", i)
		p0 := path[i]
		b.Move(AbsolutePos(p0.X), AbsolutePos(p0.Y))
	}

	if closed {
		b.Comment("Close path")
		p0 := path[0]
		b.Move(AbsolutePos(p0.X), AbsolutePos(p0.Y))
	}

	b.Up()

	b.Commentf("END DrawPath(%v, %v)", closed, path)

	return b
}

// DrawCircle draws circle on absolute (x,y) with radius r.
func (b *GCodeBuilder) DrawCircle(xImg, yImg AbsolutePos, r float32) *GCodeBuilder {
	b.Commentf("BEGIN DrawCircle(%f, %f, %f)", xImg, yImg, r)

	// 1.0: find x,y to move
	x, y := translate(xImg, yImg)
	baseX := xImg
	baseY := yImg + AbsolutePos(r)
	b.Move(baseX, baseY)
	// 1.1: do circle
	relX, relY := b.absToRel(x, y)
	b.Down()
	b.code += fmt.Sprintf("G2 I%[1]f J%[2]f ; Draw circle with center in %[1]f and %[2]f with radius %[3]f\n", relX, relY, r)
	b.Up()

	b.Commentf("END DrawCircle(%f, %f, %f)", xImg, yImg, r)
	return b
}

// DrawCircleFilled draws a filled circle.
// Make sure to set headSize before.
func (b *GCodeBuilder) DrawCircleFilled(x, y AbsolutePos, radius float32) *GCodeBuilder {
	b.Commentf("BEGIN DrawCircleFilled(%f, %f, %f)", x, y, radius)

	for r := radius; r > 0; r -= float32(b.headSize) {
		b.DrawCircle(x, y, r)
	}

	b.Commentf("END DrawCircleFilled(%f, %f, %f)", x, y, radius)

	return b
}

// DrawSector draws a sector (part of circle) on absolute (x,y) with radius r.
// start is a RADIAL angle where to start, end is a RADIAL angle where to end.
// NOTE: start/end 0 point is positive X axis. Angle is counterclockwise.
// TODO: this is not tested yet. test on e.g. .DrawSector(10,10,10,0,math.Pi/2)
func (b *GCodeBuilder) DrawSector(xImg, yImg AbsolutePos, radius float32, start, end float32) *GCodeBuilder {
	b.Commentf("BEGIN DrawSector(%f, %f, %f, %f, %f)", xImg, yImg, radius, start, end)

	// 1.0: find x,y to move
	x, y := translate(xImg, yImg)
	baseX := xImg + AbsolutePos(math.Cos(float64(start))*float64(radius))
	baseY := yImg + AbsolutePos(math.Sin(float64(start))*float64(radius))
	b.Move(baseX, baseY)
	// 1.1: find final x,y
	finalX := xImg + AbsolutePos(math.Cos(float64(end))*float64(radius))
	finalY := yImg + AbsolutePos(math.Sin(float64(end))*float64(radius))
	relFinalX, relFinalY := b.absToRel(translate(finalX, finalY))
	// 1.2: do circle
	relX, relY := b.absToRel(x, y)
	b.Down()
	b.code += fmt.Sprintf("G3 I%[1]f J%[2]f X%[3]f Y%[4]f; Draw circle with center in %[1]f and %[2]f with radius %[5]f\n", relX, relY, relFinalX, relFinalY, radius)
	b.currentX, b.currentY = translate(finalX, finalY)
	b.Up()

	b.Commentf("END DrawSector(%f, %f, %f, %f, %f)", x, y, radius, start, end)
	return b
}

// DrawRect draws rectangle from (x0, y0) to (x1, y1).
func (b *GCodeBuilder) DrawRect(x0, y0, x1, y1 AbsolutePos) *GCodeBuilder {
	b.Commentf("BEGIN DrawRect(%f, %f, %f, %f)", x0, y0, x1, y1)

	b.Move(x0, y0)
	b.Down()
	b.Move(x1, y0)
	b.Move(x1, y1)
	b.Move(x0, y1)
	b.Move(x0, y0)
	b.Up()

	b.Commentf("END DrawRect(%f, %f, %f, %f)", x0, y0, x1, y1)

	return b
}

// DrawRectFilled draws a filled rectangle.
func (b *GCodeBuilder) DrawRectFilled(x0, y0, x1, y1 AbsolutePos) *GCodeBuilder {
	b.Commentf("BEGIN DrawRectFilled(%f, %f, %f, %f)", x0, y0, x1, y1)

	delta := AbsolutePos(b.headSize)
	for x00, y00, x11, y11 := x0, y0, x1, y1; x00 < x11 || y00 < y11; x00, y00, x11, y11 = x00+delta, y00+delta, x11-delta, y11-delta {
		b.DrawRect(x00, y00, x11, y11)
	}

	b.Commentf("END DrawRectFilled(%f, %f, %f, %f)", x0, y0, x1, y1)

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
		glg.Fatalf("Absolute position must be less than %v, got %f", MaxX, x)
	case y < MinY:
		glg.Fatalf("Absolute position must be larger than %f, got %f", MinY, y)
	case y > MaxY:
		glg.Fatalf("Absolute position must be less than %v, got %f", MaxY, y)
	}

	return x, y
}

// translate converts AbsolutePos to HardwareAbsolutePos by adding MinX/Y
func translate(x, y AbsolutePos) (HardwareAbsolutePos, HardwareAbsolutePos) {
	return validateHwAbs(HardwareAbsolutePos(x+MinX), HardwareAbsolutePos(y+MinY))
}
