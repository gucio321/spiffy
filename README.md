[![Go Report Card](https://goreportcard.com/badge/github.com/gucio321/spiffy)](https://goreportcard.com/report/github.com/gucio321/spiffy)
[![GoDoc](https://pkg.go.dev/badge/github.com/gucio321/spiffy?utm_source=godoc)](https://pkg.go.dev/mod/github.com/gucio321/spiffy)

# INTRO

Spiffy is a SVG to GCode converter.

`cmd/spiffy` is intended to take a SVG file and generate a GCode file that can be used to engrave the SVG
onto a material using a CNC machine.

## Progress/Current status

- [X] Load SVG file
- [X] Parse SVG to something GO-readable
- [ ] Generate GCode from SVG
   - [ ] Paths (for now only M (Absolute Move) supported)
   - [X] Circles
   - [X] Rectangles
   - [ ] Bezier curves
   - [ ] Text

## Reference
- GCode: https://marlinfw.org/docs/gcode/G005.html
- SVG: https://developer.mozilla.org/en-US/docs/Web/SVG
