package viewer

import (
	"github.com/gucio321/spiffy/pkg/gcb"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"golang.org/x/image/colornames"
)

var _ ebiten.Game = &Viewer{}

const (
	scale  = 10.0
	startY = 600 / scale
)

var (
	borderColor = colornames.White
	travelColor = colornames.Green
	drawColor   = colornames.Red
)

// Viewer creates in NewViewer an image from gcb.GCodeBuilder and static displays it in ebiten.
type Viewer struct {
	dest *ebiten.Image
}

func NewViewer(g *gcb.GCodeBuilder) *Viewer {
	dest := ebiten.NewImage((gcb.MaxX+gcb.MinX)*scale, (gcb.MaxY+gcb.MinY)*scale)
	dest.Fill(colornames.Black)
	isDrawing := false

	/*
		ebitenutil.DrawLine(dest, gcb.MinX*scale, gcb.MinY*scale, gcb.MinX*scale, gcb.MaxY*scale, borderColor)
		ebitenutil.DrawLine(dest, gcb.MinX*scale, gcb.MaxY*scale, gcb.MaxX*scale, gcb.MaxY*scale, borderColor)
		ebitenutil.DrawLine(dest, gcb.MaxX*scale, gcb.MaxY*scale, gcb.MaxX*scale, gcb.MinY*scale, borderColor)
		ebitenutil.DrawLine(dest, gcb.MaxX*scale, gcb.MinY*scale, gcb.MinX*scale, gcb.MinY*scale, borderColor)
		currentX, currentY := float64(gcb.MinX), float64(gcb.MaxY)
	*/
	currentX, currentY := 0.0, float64(startY)

	for _, cmd := range g.Commands() {
		switch cmd.Code {
		case "G0":
			if _, ok := cmd.Args["Z"]; ok { // we assume this is up/down command for now
				isDrawing = !isDrawing
			}
			if _, ok := cmd.Args["X"]; ok {
				newX := currentX + float64(cmd.Args["X"])
				newY := currentY - float64(cmd.Args["Y"]) // this is because of 0,0 difference
				c := drawColor
				if !isDrawing {
					c = travelColor
				}

				ebitenutil.DrawLine(dest, currentX*scale, currentY*scale, newX*scale, newY*scale, c)

				currentX, currentY = newX, newY
			}
		}
	}

	return &Viewer{
		dest: dest,
	}
}

func (v *Viewer) Update() error {
	return nil
}

func (v *Viewer) Draw(screen *ebiten.Image) {
	screen.DrawImage(v.dest, nil)
}

func (v *Viewer) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// return v.dest.Bounds().Dx(), v.dest.Bounds().Dy()
	return outsideWidth, outsideHeight
}
