package spiffy

import (
	"errors"
	"image"
	"strconv"
	"strings"
)

type Spiffy struct {
	scale float32
	Defs  Defs  `xml:"defs"`
	Graph Graph `xml:"g"`
}

type Defs struct {
	Rects   []Rect           `xml:"rect"`
	Linears []LinearGradient `xml:"linearGradient"`
}

type Rect struct {
	X  float32 `xml:"x,attr"`
	Y  float32 `xml:"y,attr"`
	W  float32 `xml:"width,attr"`
	H  float32 `xml:"height,attr"`
	Id string  `xml:"id,attr"`
}

type LinearGradient struct {
	Id    string  `xml:"id,attr"`
	X1    float32 `xml:"x1,attr"`
	Y1    float32 `xml:"y1,attr"`
	X2    float32 `xml:"x2,attr"`
	Y2    float32 `xml:"y2,attr"`
	Stops []Stop  `xml:"stop"` // this programm wil not use that but I'll keep it just in case
}

type Stop struct {
	Offset float32 `xml:"offset,attr"`
	Style  string  `xml:"style,attr"`
}

type Graph struct {
	Circles []Circle    `xml:"circle"`
	Paths   []Path      `xml:"path"`
	Rects   []GraphRect `xml:"rect"`
	Texts   []Text      `xml:"text"`
}

type Circle struct {
	Cx   float32 `xml:"cx,attr"`
	Cy   float32 `xml:"cy,attr"`
	R    float32 `xml:"r,attr"`
	Fill string  `xml:"fill,attr"`
}

type Path struct {
	Style string `xml:"style,attr"`
	D     string `xml:"d,attr"`
	ID    string `xml:"id,attr"`
}

type GraphRect struct {
	Style string  `xml:"style,attr"`
	ID    string  `xml:"id,attr"`
	X     float32 `xml:"x,attr"`
	Y     float32 `xml:"y,attr"`
	W     float32 `xml:"width,attr"`
	H     float32 `xml:"height,attr"`
}

type Text struct {
	ID    string  `xml:"id,attr"`
	Tspan []Tspan `xml:"tspan"`
}

type Tspan struct {
	X      float32 `xml:"x,attr"`
	Y      float32 `xml:"y,attr"`
	ID     string  `xml:"id,attr"`
	Tspan2 Tspan2  `xml:"tspan"`
}

type Tspan2 struct {
	Style string `xml:"style,attr"`
	Text  string `xml:",chardata"`
}

func (s *Spiffy) Scale(scale float32) *Spiffy {
	s.scale = scale
	return s
}

// GCode returns single-purpose GCode for our project.
func (s *Spiffy) GCode() (string, error) {
	builder := NewGCodeBuilder()

	// 1.0: draw paths
	for _, line := range s.Graph.Paths {
		txt := line.D
		txts := strings.Split(txt, " ")
		if txts[0] != "M" {
			return "", errors.New("Unexpected paths.D prefix; Not implemented")
		}

		paths := make([]image.Point, 0, len(txts)-1)

		for _, t := range txts[1:] {
			parts := strings.Split(t, ",")
			if len(parts) != 2 {
				return "", errors.New("Unexpected paths.D parts; Not implemented")
			}

			xStr, yStr := parts[0], parts[1]
			// parse floats
			x, err := strconv.ParseFloat(xStr, 32)
			if err != nil {
				return "", err
			}

			y, err := strconv.ParseFloat(yStr, 32)
			if err != nil {
				return "", err
			}

			paths = append(paths, image.Point{X: int(x * float64(s.scale)), Y: int(y * float64(s.scale))}) // TODO: loses precision; use custom thing instead of image.Point
		}

		builder.DrawPath(true, paths...)
		builder.Separator()
	}

	// 2.0: draw circles
	for _, c := range s.Graph.Circles {
		builder.DrawCircleFilled(AbsolutePos(c.Cx*s.scale), AbsolutePos(c.Cy*s.scale), (c.R * s.scale))
		builder.Separator()
	}

	result := builder.String()
	return result, nil
}
