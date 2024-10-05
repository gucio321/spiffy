package gcb

// GCode represents a gcode (e.g. G0, G1, G91)
type GCode string

// list of gcodes. See https://marlinfw.org/docs/gcode/G005.html
// We point out only codes used in this project.
const (
	// G0 is a move command
	G0 GCode = "G0"
	// G1 is a move command too (ref does not point the difference)
	G1 GCode = "G1"
	// G2 is a clockwise arc move
	G2 GCode = "G2"
	// G5 is a cubic B-spline move
	G5 GCode = "G5"

	GCodeMove        = G0
	GCodeArc         = G2
	GCodeBezierCubic = G5
)
