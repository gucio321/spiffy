# INTRO

Spiffy is a SVG to GCode converter.

`cmd/spiffy` is intended to take a SVG file and generate a GCode file that can be used to engrave the SVG
onto a material using a CNC machine.

## Progress/Current status

- [X] Load SVG file
- [X] Parse SVG to something GO-readable
- [ ] Generate GCode from SVG
   - [X] Paths
   - [X] Circles
   - [X] Rectangles
   - [ ] Bezier curves
   - [ ] Text

## Reference
- GCode: https://marlinfw.org/docs/gcode/G005.html
