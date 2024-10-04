package viewer

import (
	"image"

	"github.com/gucio321/spiffy/pkg/gcb"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"golang.org/x/image/colornames"
)

var _ ebiten.Game = &Viewer{}

const (
	baseScale = 7.0
	startY    = 600 / baseScale
)

var (
	borderColor = colornames.White
	travelColor = colornames.Green
	drawColor   = colornames.Red
)

// Viewer creates in NewViewer an image from gcb.GCodeBuilder and static displays it in ebiten.
type Viewer struct {
	scale   float64
	gcode   *gcb.GCodeBuilder
	current *ebiten.Image
}

func NewViewer(g *gcb.GCodeBuilder) *Viewer {
	result := &Viewer{
		scale: 1,
		gcode: g,
	}

	result.current = result.render()
	return result
}

func (g *Viewer) render() *ebiten.Image {
	scale := g.scale * baseScale
	dest := ebiten.NewImage(int((gcb.MaxX+gcb.MinX)*scale), int(startY*scale))
	dest.Fill(colornames.Black)
	isDrawing := false

	/*
		ebitenutil.DrawLine(dest, gcb.MinX*scale, gcb.MinY*scale, gcb.MinX*scale, gcb.MaxY*scale, borderColor)
		ebitenutil.DrawLine(dest, gcb.MinX*scale, gcb.MaxY*scale, gcb.MaxX*scale, gcb.MaxY*scale, borderColor)
		ebitenutil.DrawLine(dest, gcb.MaxX*scale, gcb.MaxY*scale, gcb.MaxX*scale, gcb.MinY*scale, borderColor)
		ebitenutil.DrawLine(dest, gcb.MaxX*scale, gcb.MinY*scale, gcb.MinX*scale, gcb.MinY*scale, borderColor)
		currentX, currentY := float64(gcb.MinX), float64(gcb.MaxY)
	*/
	//ebitenutil.DrawLine(dest, (gcb.MaxX-gcb.MinX)*scale, (gcb.MaxY-gcb.MinY)*scale, gcb.MinX*scale, gcb.MaxY*scale, borderColor)
	ebitenutil.DrawLine(dest,
		(gcb.MaxX-gcb.MinX)*scale, startY*scale,
		(gcb.MaxX-gcb.MinX)*scale, (startY-(gcb.MaxY-gcb.MinY))*scale,
		borderColor)

	ebitenutil.DrawLine(dest,
		0*scale, (startY-(gcb.MaxY-gcb.MinY))*scale,
		(gcb.MaxX-gcb.MinX)*scale, (startY-(gcb.MaxY-gcb.MinY))*scale,
		borderColor)

	currentX, currentY := 0.0, float64(startY)

	for _, cmd := range g.gcode.Commands() {
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

	return dest
}

func (v *Viewer) Update() error {
	_, wheelY := ebiten.Wheel()
	v.scale += wheelY * 0.1
	if v.scale < 1 {
		v.scale = 1
	}

	return nil
}

func (v *Viewer) Draw(screen *ebiten.Image) {
	const w, h = 800, 600
	mouseX, mouseY := ebiten.CursorPosition()
	// negative check lol
	if mouseX < 0 {
		mouseX = 0
	}

	if mouseY < 0 {
		mouseY = 0
	}

	renderable := v.current.SubImage(image.Rect(
		int((v.scale-1)*float64(mouseX)), int((v.scale-1)*float64(mouseY)),
		int(w+(v.scale-1)*float64(mouseX)), int(h+(v.scale-1)*float64(mouseY))))

	if renderable.Bounds().Dx() == 0 || renderable.Bounds().Dy() == 0 {
		renderable = v.current
	}

	geom := ebiten.GeoM{}
	geom.Scale(v.scale, v.scale)
	screen.DrawImage(ebiten.NewImageFromImage(renderable),
		&ebiten.DrawImageOptions{
			GeoM: geom,
		})
}

func (v *Viewer) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// return v.dest.Bounds().Dx(), v.dest.Bounds().Dy()
	return outsideWidth, outsideHeight
}
