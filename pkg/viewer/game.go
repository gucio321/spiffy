package viewer

import (
	"fmt"
	"image"

	"github.com/AllenDang/cimgui-go/backend"
	ebitenbackend "github.com/AllenDang/cimgui-go/backend/ebiten-backend"
	"github.com/AllenDang/cimgui-go/imgui"
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
	scale        float64
	gcode        *gcb.GCodeBuilder
	current      *ebiten.Image
	imgui        *ebitenbackend.EbitenBackend
	showMoves    bool
	showPrinting bool
}

func NewViewer(g *gcb.GCodeBuilder) *Viewer {
	ebitenBackend := ebitenbackend.NewEbitenBackend()
	backend.CreateBackend(ebitenBackend)
	ebitenBackend.CreateWindow("GCode Viewer", 800, 600)

	result := &Viewer{
		scale:        1,
		gcode:        g,
		imgui:        ebitenBackend,
		showMoves:    true,
		showPrinting: true,
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

				if !((isDrawing && !g.showPrinting) || (!isDrawing && !g.showMoves)) {
					ebitenutil.DrawLine(dest, currentX*scale, currentY*scale, newX*scale, newY*scale, c)
				}

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

	// render cimgui
	v.imgui.BeginFrame()
	imgui.SetNextWindowSizeV(imgui.Vec2{250, 110}, imgui.CondAlways)
	imgui.SetNextWindowPos(imgui.Vec2{0, 0})
	imgui.BeginV("Settings", nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoTitleBar) //|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsNoFocusOnAppearing|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoDocking|imgui.WindowFlagsNoNav|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoInputs|imgui.WindowFlagsNoMouseInputs|imgui.WindowFlagsNoMouseInputsOnChildren|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoDocking|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoFocusOnAppearing|imgui.WindowFlagsNoMouseInputsOnChildren|imgui.WindowFlagsNoMouseInputs|imgui.WindowFlagsNoInputs|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoDocking|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoFocusOnAppearing|imgui.WindowFlagsNoMouseInputsOnChildren|imgui.WindowFlagsNoMouseInputs|imgui.WindowFlagsNoInputs|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs)

	imgui.Text(fmt.Sprintf(`use scrool to zoom in/out
Scale: %.2f
`, v.scale))
	imgui.PushStyleColorVec4(imgui.ColText, imgui.Vec4{0, 1, 0, 1})

	if imgui.Checkbox("Show Moves (without drawing)", &v.showMoves) {
		v.current = v.render()
	}

	imgui.PopStyleColor()

	imgui.PushStyleColorVec4(imgui.ColText, imgui.Vec4{1, 0, 0, 1})

	if imgui.Checkbox("Show Drawing", &v.showPrinting) {
		v.current = v.render()
	}

	imgui.PopStyleColor()

	imgui.End()
	v.imgui.EndFrame()

	return nil
}

func (v *Viewer) Draw(screen *ebiten.Image) {
	screen.Fill(colornames.Blue)

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

	v.imgui.Draw(screen)
}

func (v *Viewer) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// return v.dest.Bounds().Dx(), v.dest.Bounds().Dy()
	v.imgui.Layout(outsideWidth, outsideHeight)
	return outsideWidth, outsideHeight
}