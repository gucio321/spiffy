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
	"github.com/kpango/glg"
	"golang.org/x/image/colornames"
)

var _ ebiten.Game = &Viewer{}

const (
	screenW, screenH = 800, 600
	baseScale        = screenH/float64(gcb.MaxY-gcb.MinY) - .5
	startY           = 600 / baseScale
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
	lockedX, lockedY int
	isMouseOverUI    bool
	w, h             int
	axesModifiers    [2]int // x, y. supposed to be 1 or -1 for mirroring.
	xMirror, yMirror bool
}

func NewViewer(g *gcb.GCodeBuilder) *Viewer {
	fmt.Println(baseScale)
	ebitenBackend := ebitenbackend.NewEbitenBackend()
	backend.CreateBackend(ebitenBackend)
	ebitenBackend.CreateWindow("GCode Viewer", screenW, screenH)

	result := &Viewer{
		scale:           1,
		gcode:           g,
		imgui:           ebitenBackend,
		showMoves:       true,
		showPrinting:    true,
		showStateChange: true,
		cmdRange:        [2]int32{0, int32(len(g.Commands()))},
		playTickMs:      100,
		w:               screenW,
		h:               screenH,
		axesModifiers:   [2]int{1, 1},
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

	scale := baseScale
	dest := ebiten.NewImage(g.w, g.h)
	dest.Fill(colornames.Black)
	isDrawing := false

	ebitenutil.DrawLine(dest,
		(gcb.MaxX-gcb.MinX)*scale, startY*scale,
		(gcb.MaxX-gcb.MinX)*scale, (startY-(gcb.MaxY-gcb.MinY))*scale,
		borderColor)

	ebitenutil.DebugPrintAt(dest,
		fmt.Sprintf("Min Y: %d, Max Y: %d, H: %d", gcb.MinY, gcb.MaxY, (gcb.MaxY-gcb.MinY)),
		5+int((gcb.MaxX-gcb.MinX)*scale), int((startY*scale+(startY-(gcb.MaxY-gcb.MinY))*scale)/2),
	)

	ebitenutil.DrawLine(dest,
		0*scale, (startY-(gcb.MaxY-gcb.MinY))*scale,
		(gcb.MaxX-gcb.MinX)*scale, (startY-(gcb.MaxY-gcb.MinY))*scale,
		borderColor)

	ebitenutil.DebugPrintAt(dest,
		fmt.Sprintf("Min X: %d, Max X: %d, W: %d", gcb.MinX, gcb.MaxX, (gcb.MaxX-gcb.MinX)),
		int((gcb.MaxX-gcb.MinX)*scale/2), int((startY-(gcb.MaxY-gcb.MinY))*scale)-20,
	)

	var currentX, currentY float64

	switch g.axesModifiers[0] {
	case 1:
		currentX = float64(gcb.BaseX - gcb.MinX)
	case -1:
		currentX = float64(gcb.MaxX-gcb.MinX) - float64(gcb.BaseX-gcb.MinX)
	}

	switch g.axesModifiers[1] {
	case 1:
		currentY = float64(startY) - (gcb.BaseY - gcb.MinY)
	case -1:
		currentY = float64(gcb.MaxY-gcb.MinY) - (float64(startY) - (gcb.BaseY - gcb.MinY))
	}

	for _, cmd := range g.gcode.Commands()[g.cmdRange[0]:endFrame] {
		switch cmd.Code {
		case "G0":
			g.code += cmd.String(true, true) + "\n"
			if _, ok := cmd.Args["Z"]; ok { // we assume this is up/down command for now
				if g.showStateChange {
					ebitenutil.DrawCircle(dest, currentX*scale, currentY*scale, 2, stateChangeColor)
				}

				// also primitive (was !isDrawing earlier), but at least handles multiple Z commands
				isDrawing = cmd.Args["Z"] < 0
			}

			_, xChange := cmd.Args["X"]
			_, yChange := cmd.Args["Y"]
			if xChange || yChange {
				newX := currentX + float64(cmd.Args["X"])*float64(g.axesModifiers[0])
				newY := currentY - float64(cmd.Args["Y"])*float64(g.axesModifiers[1]) // this is because of 0,0 difference

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
	imgui.SetNextWindowSizeV(imgui.Vec2{250, 155}, imgui.CondAlways)
	imgui.SetNextWindowPos(imgui.Vec2{0, 0})
	imgui.BeginV("Settings", nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove) //|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsNoFocusOnAppearing|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoDocking|imgui.WindowFlagsNoNav|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoInputs|imgui.WindowFlagsNoMouseInputs|imgui.WindowFlagsNoMouseInputsOnChildren|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoDocking|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoFocusOnAppearing|imgui.WindowFlagsNoMouseInputsOnChildren|imgui.WindowFlagsNoMouseInputs|imgui.WindowFlagsNoInputs|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoNavFocusOnAppearing|imgui.WindowFlagsNoDocking|imgui.WindowFlagsNoBackground|imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoFocusOnAppearing|imgui.WindowFlagsNoMouseInputsOnChildren|imgui.WindowFlagsNoMouseInputs|imgui.WindowFlagsNoInputs|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoCollapse|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoBringToFrontOnFocus|imgui.WindowFlagsNoNavFocus|imgui.WindowFlagsNoNavInputs)

	imgui.Text(fmt.Sprintf(`use scrool to zoom in/out, click+move to move
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

		imgui.Text("Mirroring:")
		imgui.SameLine()
		if imgui.Checkbox("X Mirror", &v.xMirror) {
			if v.xMirror {
				v.axesModifiers[1] = -1
			} else {
				v.axesModifiers[1] = 1
			}

			v.current = v.render()
		}

		imgui.SameLine()
		if imgui.Checkbox("Y Mirror", &v.yMirror) {
			if v.yMirror {
				v.axesModifiers[0] = -1
			} else {
				v.axesModifiers[0] = 1
			}

			v.current = v.render()
		}

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

		if imgui.TreeNodeExStr("Source Code (GCode)") {
			if imgui.BeginChildStrV("code", imgui.Vec2{-1, 300}, 0, imgui.WindowFlagsHorizontalScrollbar) {
				v.isMouseOverUI = v.isMouseOverUI || imgui.IsWindowHovered()
				imgui.Text(v.code)
				if v.isPlaying {
					imgui.SetScrollYFloat(imgui.ScrollMaxY())
				}
				imgui.EndChild()
			}

			imgui.TreePop()
		}

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

	mouseX, mouseY := int(v.lockedX), int(v.lockedY)

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !v.isMouseOverUI {
		mouseX, mouseY = ebiten.CursorPosition()

		// negative check lol
		if mouseX < 0 {
			mouseX = 0
		}

		if mouseY < 0 {
			mouseY = 0
		}

		if mouseX > v.w {
			mouseX = v.w
		}

		if mouseY > v.h {
			mouseY = v.h
		}

		v.lockedX, v.lockedY = mouseX, mouseY
	}

	rect := image.Rect(
		int((v.scale-1)*float64(mouseX)/v.scale),
		int((v.scale-1)*float64(mouseY)/v.scale),
		int(float64(v.w)+(v.scale-1)*float64(mouseX)),
		int(float64(v.h)+(v.scale-1)*float64(mouseY)),
	)

	renderable := v.current.SubImage(rect)

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
	v.w, v.h = outsideWidth, outsideHeight
	// return v.dest.Bounds().Dx(), v.dest.Bounds().Dy()
	v.imgui.Layout(outsideWidth, outsideHeight)
	return outsideWidth, outsideHeight
}
