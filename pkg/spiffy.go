package spiffy

type Spiffy struct {
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
