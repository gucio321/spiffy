[![Go Report Card](https://goreportcard.com/badge/github.com/gucio321/spiffy)](https://goreportcard.com/report/github.com/gucio321/spiffy)
[![GoDoc](https://pkg.go.dev/badge/github.com/gucio321/spiffy?utm_source=godoc)](https://pkg.go.dev/mod/github.com/gucio321/spiffy)

# INTRO

Spiffy is a SVG to GCode converter.

`cmd/spiffy` is intended to take a SVG file and generate a GCode file that can be used to engrave the SVG
onto a material using a CNC machine.

## Requirements

Since our svg parser is not perfect and only supports paths, we use
`Inkscape` to convert any SVG to the supported format.

## Progress/Current status

- [X] Load SVG file
- [X] Parse SVG to something GO-readable
- [ ] Generate GCode from SVG
   - [X] Paths
   - [X] Circles
   - [X] Rectangles
   - [X] Text (if converted to paths via ikscape)

## Reference
- GCode: https://marlinfw.org/docs/gcode/G005.html
- SVG: https://developer.mozilla.org/en-US/docs/Web/SVG
- testing tool: https://nraynaud.github.io/webgcode/
- Inkscape API ref: https://wiki.inkscape.org/wiki/Action


# Legal notes.

`cmd/spiffy/AGH.svg` is a trademark/registered trademark of [AGH University of Science and Technology](https://www.agh.edu.pl). It is used here for educational purposes only.

The project is licensed under the (attached) [MIT License](LICENSE).
