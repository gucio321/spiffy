package viewer

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"github.com/AllenDang/cimgui-go/backend"
	ebitenbackend "github.com/AllenDang/cimgui-go/backend/ebiten-backend"
	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/gucio321/spiffy/pkg/gcb"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/kpango/glg"
	"golang.org/x/image/colornames"
)

var _ ebiten.Game = &Viewer{}

const (
	baseScale = 7.0
	startY    = 600 / baseScale
)

var (
	borderColor      = colornames.White
	travelColor      = colornames.Green
	stateChangeColor = colornames.Yellow
	drawColor        = colornames.Red
)

// Viewer creates in NewViewer an image from gcb.GCodeBuilder and static displays it in ebiten.
type Viewer struct {
	scale            float64
	gcode            *gcb.GCodeBuilder
	code             string
	current          *ebiten.Image
	imgui            *ebitenbackend.EbitenBackend
	showMoves        bool
	showPrinting     bool
	showStateChange  bool
	showAdvanced     bool
	cmdRange         [2]int32
	playTickMs       int32
	isPlaying        bool
	currentFrame     int
	t                time.Time
	locked           bool
	lockedX, lockedY float64
	isMouseOverUI    bool
}

func NewViewer(g *gcb.GCodeBuilder) *Viewer {
	ebitenBackend := ebitenbackend.NewEbitenBackend()
	backend.CreateBackend(ebitenBackend)
	ebitenBackend.CreateWindow("GCode Viewer", 800, 600)

	result := &Viewer{
		scale:           1,
		gcode:           g,
		imgui:           ebitenBackend,
		showMoves:       true,
		showPrinting:    true,
		showStateChange: true,
		cmdRange:        [2]int32{0, int32(len(g.Commands()))},
		playTickMs:      100,
	}

	result.current = result.render()
	return result
}

func (g *Viewer) render() *ebiten.Image {
	g.code = ""

	endFrame := g.cmdRange[1]
	if g.isPlaying {
		endFrame = int32(g.currentFrame)
	}

	scale := g.scale * baseScale
	dest := ebiten.NewImage(int((gcb.MaxX)*scale), int(startY*scale))
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

	for _, cmd := range g.gcode.Commands()[g.cmdRange[0]:endFrame] {
		switch cmd.Code {
		case "G0":
			g.code += cmd.String(true, true) + "\n"
			if _, ok := cmd.Args["Z"]; ok { // we assume this is up/down command for now
				if g.showStateChange {
					ebitenutil.DrawCircle(dest, currentX*scale, currentY*scale, 2, stateChangeColor)
				}

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
		case "":
			g.code += cmd.String(true, true) + "\n"
		default:
			glg.Warnf("Unknown command: %s", cmd.Code)
		}
	}

	return dest
}

func (v *Viewer) Update() error {
	var wheelY float64
	if !v.isMouseOverUI {
		_, wheelY = ebiten.Wheel()
	}

	v.scale += wheelY * 0.1
	if v.scale < 1 {
		v.scale = 1
	}

	// render cimgui
	v.imgui.BeginFrame()
	imgui.SetNextWindowSizeV(imgui.Vec2{250, 135}, imgui.CondAlways)
	imgui.SetNextWindowPos(imgui.Vec2{0, 0})
	imgui.BeginV("Settings", nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoTitleBar) //|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsNoFocusOnAppearing|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoDocking|imgui.WindowFlagsNoNav|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoInputs|imgui.WindowFlagsNoMouseInputs|imgui.WindowFlagsNoMouseInputsOnChildren|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoDocking|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoFocusOnAppearing|imgui.WindowFlagsNoMouseInputsOnChildren|imgui.WindowFlagsNoMouseInputs|imgui.WindowFlagsNoInputs|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoDocking|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoFocusOnAppearing|imgui.WindowFlagsNoMouseInputsOnChildren|imgui.WindowFlagsNoMouseInputs|imgui.WindowFlagsNoInputs|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs)

	imgui.Text(fmt.Sprintf(`use scrool to zoom in/out, space to freez, click+move to move
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

	imgui.PushStyleColorVec4(imgui.ColText, imgui.Vec4{1, 1, 0, 1})

	if imgui.Checkbox("Show State changes (start/stop drawing)", &v.showStateChange) {
		v.current = v.render()
	}

	imgui.PopStyleColor()

	imgui.Checkbox("Advanced", &v.showAdvanced)

	imgui.End()

	v.isMouseOverUI = false
	if v.showAdvanced {
		imgui.Begin("Advanced GCode")
		v.isMouseOverUI = v.isMouseOverUI || imgui.IsWindowHovered()
		imgui.BeginDisabledV(v.isPlaying)
		imgui.Text("Command Range:")

		imgui.PushItemWidth(80)
		if imgui.SliderInt("##start", &v.cmdRange[0], 0, v.cmdRange[1]) {
			v.current = v.render()
		}

		imgui.SameLine()

		if imgui.InputInt("##startText", &v.cmdRange[0]) {
			if v.cmdRange[0] < 0 {
				v.cmdRange[0] = 0
			}

			if v.cmdRange[0] > v.cmdRange[1] {
				v.cmdRange[0] = v.cmdRange[1]
			}

			v.current = v.render()
		}

		imgui.PopItemWidth()

		imgui.PushItemWidth(80)
		if imgui.SliderInt("##end", &v.cmdRange[1], v.cmdRange[0], int32(len(v.gcode.Commands())-1)) {
			v.current = v.render()
		}

		imgui.SameLine()

		if imgui.InputInt("##endText", &v.cmdRange[1]) {
			if v.cmdRange[1] > int32(len(v.gcode.Commands())-1) {
				v.cmdRange[1] = int32(len(v.gcode.Commands()) - 1)
			}

			if v.cmdRange[1] < v.cmdRange[0] {
				v.cmdRange[1] = v.cmdRange[0]
			}

			v.current = v.render()
		}

		imgui.PopItemWidth()

		imgui.Text("Player:")
		imgui.SameLine()
		imgui.PushItemWidth(80)
		if imgui.InputInt("##player", &v.playTickMs) {
			if v.playTickMs < 1 {
				v.playTickMs = 1
			}
		}

		imgui.PopItemWidth()

		imgui.SameLine()
		imgui.Text("ms")
		imgui.EndDisabled()
		imgui.SameLine()
		if v.isPlaying {
			imgui.PushStyleColorVec4(imgui.ColText, imgui.Vec4{1, 0, 0, 1})
			if imgui.Button("Stop") {
				v.isPlaying = false
				v.current = v.render() // rerender to the default view
			}

			imgui.PopStyleColor()
		} else {
			imgui.PushStyleColorVec4(imgui.ColText, imgui.Vec4{0, 1, 0, 1})
			if imgui.Button("Start") {
				v.isPlaying = true
				v.t = time.Now()
				v.currentFrame = int(v.cmdRange[0])
			}

			imgui.PopStyleColor()
		}

		if imgui.TreeNodeExStr("Source Code (GCode)") {
			if imgui.BeginChildStrV("code", imgui.Vec2{-1, 300}, 0, imgui.WindowFlagsHorizontalScrollbar) {
				v.isMouseOverUI = v.isMouseOverUI || imgui.IsWindowHovered()
				imgui.Text(v.code)
				imgui.EndChild()
			}

			imgui.TreePop()
		}

		if imgui.Button("Export current frame") {
			filename := "frame.png"
			b := bytes.NewBufferString("")
			fmt.Println("start encode")
			png.Encode(b, v.current)
			fmt.Println("end encode")
			if err := os.WriteFile(filename, b.Bytes(), 0644); err != nil {
				glg.Errorf("Error while exporting frame: %v", err)
			}

			glg.Infof("Current frame was exported as %v", filename)
		}

		imgui.End()
	}

	v.imgui.EndFrame()

	// now handle player
	if v.isPlaying {
		// first check if should not stop
		if v.currentFrame >= int(v.cmdRange[1]) {
			v.isPlaying = false
		}

		delta := time.Since(v.t)
		if delta >= time.Duration(time.Duration(v.playTickMs)*time.Millisecond) {
			v.t = time.Now()
			v.currentFrame++
			v.current = v.render()
		}
	}

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

	if v.locked {
		mouseX, mouseY = int(v.lockedX), int(v.lockedY)
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

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		v.locked = !v.locked
		v.lockedX, v.lockedY = float64(mouseX), float64(mouseY)
	}
}

func (v *Viewer) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// return v.dest.Bounds().Dx(), v.dest.Bounds().Dy()
	v.imgui.Layout(outsideWidth, outsideHeight)
	return outsideWidth, outsideHeight
}
