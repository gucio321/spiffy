package viewer

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"sync"
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

func (v *Viewer) baseScale() float64 {
	return screenH/float64(v.gcode.Workspace().MaxY-v.gcode.Workspace().MinY) - .5
}

func (v *Viewer) startY() float64 {
	// TODO: what the fuck is 600?
	return 600 / v.baseScale()
}

const (
	screenW, screenH = 800, 600
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
	Y                struct {
		Min, Max, Delta int
	}
	rendering         *sync.WaitGroup
	isRendering       bool
	renderingProgress float32
}

func NewViewer(g *gcb.GCodeBuilder) *Viewer {
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
		rendering:       &sync.WaitGroup{},
	}

	// claculate Y stats
	current := 0
	for _, cmd := range g.Commands() {
		if cmd.Code != "G0" {
			continue
		}

		z, ok := cmd.Args["Z"]
		if !ok {
			continue
		}

		current += int(z)
		if current < result.Y.Min {
			result.Y.Min = current
		}

		if current > result.Y.Max {
			result.Y.Max = current
		}
	}

	result.Y.Delta = result.Y.Max - result.Y.Min

	result.current = result.render()
	return result
}

func (v *Viewer) render() *ebiten.Image {
	// here we make a render queue. No other render() will start as long as this is running.
	v.rendering.Wait()
	v.rendering.Add(1)
	v.isRendering = true

	v.code = ""

	endFrame := v.cmdRange[1]
	if v.isPlaying {
		endFrame = int32(v.currentFrame)
	}

	scale := v.baseScale()
	dest := ebiten.NewImage(v.w, v.h)
	dest.Fill(colornames.Black)
	isDrawing := false

	ebitenutil.DrawLine(dest,
		float64(v.gcode.Workspace().MaxX-v.gcode.Workspace().MinX)*scale, v.startY()*scale,
		float64(v.gcode.Workspace().MaxX-v.gcode.Workspace().MinX)*scale, float64(v.startY()-float64(v.gcode.Workspace().MaxY-v.gcode.Workspace().MinY))*scale,
		borderColor)

	ebitenutil.DebugPrintAt(dest,
		fmt.Sprintf("Min Y: %d, Max Y: %d, H: %d",
			v.gcode.Workspace().MinY, v.gcode.Workspace().MaxY, (v.gcode.Workspace().MaxY-v.gcode.Workspace().MinY)),
		5+int(float64(v.gcode.Workspace().MaxX-v.gcode.Workspace().MinX)*scale), int(float64(v.startY()*scale+(v.startY()-float64(v.gcode.Workspace().MaxY-v.gcode.Workspace().MinY))*scale)/2),
	)

	ebitenutil.DrawLine(dest,
		0*scale, float64(v.startY()-float64(v.gcode.Workspace().MaxY-v.gcode.Workspace().MinY))*scale,
		float64(v.gcode.Workspace().MaxX-v.gcode.Workspace().MinX)*scale, (v.startY()-float64(v.gcode.Workspace().MaxY-v.gcode.Workspace().MinY))*scale,
		borderColor)

	ebitenutil.DebugPrintAt(dest,
		fmt.Sprintf("Min X: %d, Max X: %d, W: %d", v.gcode.Workspace().MinX, v.gcode.Workspace().MaxX, (v.gcode.Workspace().MaxX-v.gcode.Workspace().MinX)),
		int(float64(v.gcode.Workspace().MaxX-v.gcode.Workspace().MinX)*scale/2), int((v.startY()-float64(v.gcode.Workspace().MaxY-v.gcode.Workspace().MinY))*scale)-20,
	)

	var currentX, currentY float64

	switch v.axesModifiers[0] {
	case 1:
		currentX = float64(gcb.BaseX - v.gcode.Workspace().MinX)
	case -1:
		currentX = float64(v.gcode.Workspace().MaxX-v.gcode.Workspace().MinX) - float64(gcb.BaseX-v.gcode.Workspace().MinX)
	}

	switch v.axesModifiers[1] {
	case 1:
		currentY = float64(v.startY()) - float64(gcb.BaseY-v.gcode.Workspace().MinY)
	case -1:
		currentY = float64(v.gcode.Workspace().MaxY-v.gcode.Workspace().MinY) - (float64(v.startY()) - float64(gcb.BaseY-v.gcode.Workspace().MinY))
	}

	currentZ := 0
	go func() {
		for i, cmd := range v.gcode.Commands()[v.cmdRange[0]:endFrame] {
			v.renderingProgress = float32(i) / float32(endFrame-v.cmdRange[0])
			switch cmd.Code {
			case "G0":
				v.code += cmd.String(true, true) + "\n"
				if _, ok := cmd.Args["Z"]; ok { // we assume this is up/down command for now
					if v.showStateChange {
						ebitenutil.DrawCircle(dest, currentX*scale, currentY*scale, 2, stateChangeColor)
					}

					currentZ += int(cmd.Args["Z"])
				}

				_, xChange := cmd.Args["X"]
				_, yChange := cmd.Args["Y"]
				if xChange || yChange {
					newX := currentX + float64(cmd.Args["X"])*float64(v.axesModifiers[0])
					newY := currentY - float64(cmd.Args["Y"])*float64(v.axesModifiers[1]) // this is because of 0,0 difference

					x := 7 * float64(currentZ-v.Y.Min) / float64(v.Y.Delta)
					x = x - math.Floor(x)
					c := GreenToRedHSV(x)

					if !((isDrawing && !v.showPrinting) || (!isDrawing && !v.showMoves)) {
						ebitenutil.DrawLine(dest, currentX*scale, currentY*scale, newX*scale, newY*scale, c)
					}

					currentX, currentY = newX, newY
				}
			case "":
				v.code += cmd.String(true, true) + "\n"
			default:
				glg.Warnf("Unknown command: %s", cmd.Code)
			}
		}

		v.rendering.Done()
		v.isRendering = false
	}()

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
		go func() { v.current = v.render() }()
	}

	imgui.PopStyleColor()

	imgui.PushStyleColorVec4(imgui.ColText, imgui.Vec4{1, 0, 0, 1})

	if imgui.Checkbox("Show Drawing", &v.showPrinting) {
		go func() { v.current = v.render() }()
	}

	imgui.PopStyleColor()

	imgui.PushStyleColorVec4(imgui.ColText, imgui.Vec4{1, 1, 0, 1})

	if imgui.Checkbox("Show State changes (start/stop drawing)", &v.showStateChange) {
		go func() { v.current = v.render() }()
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

			go func() { v.current = v.render() }()
		}

		imgui.SameLine()
		if imgui.Checkbox("Y Mirror", &v.yMirror) {
			if v.yMirror {
				v.axesModifiers[0] = -1
			} else {
				v.axesModifiers[0] = 1
			}

			go func() { v.current = v.render() }()
		}

		imgui.Text("Command Range:")

		imgui.PushItemWidth(80)
		if imgui.SliderInt("##start", &v.cmdRange[0], 0, v.cmdRange[1]) {
			go func() { v.current = v.render() }()
		}

		imgui.SameLine()

		if imgui.InputInt("##startText", &v.cmdRange[0]) {
			if v.cmdRange[0] < 0 {
				v.cmdRange[0] = 0
			}

			if v.cmdRange[0] > v.cmdRange[1] {
				v.cmdRange[0] = v.cmdRange[1]
			}

			go func() { v.current = v.render() }()
		}

		imgui.PopItemWidth()

		imgui.PushItemWidth(80)
		if imgui.SliderInt("##end", &v.cmdRange[1], v.cmdRange[0], int32(len(v.gcode.Commands())-1)) {
			go func() { v.current = v.render() }()
		}

		imgui.SameLine()

		if imgui.InputInt("##endText", &v.cmdRange[1]) {
			if v.cmdRange[1] > int32(len(v.gcode.Commands())-1) {
				v.cmdRange[1] = int32(len(v.gcode.Commands()) - 1)
			}

			if v.cmdRange[1] < v.cmdRange[0] {
				v.cmdRange[1] = v.cmdRange[0]
			}

			go func() { v.current = v.render() }()
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
				go func() { v.current = v.render() }()
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
			glg.Info("encoding started")
			png.Encode(b, v.current)
			glg.Info("encoding finished")
			if err := os.WriteFile(filename, b.Bytes(), 0644); err != nil {
				glg.Errorf("Error while exporting frame: %v", err)
			}

			glg.Infof("Current frame was exported as %v", filename)
		}

		imgui.End()
	}

	if v.isRendering {
		imgui.SetNextWindowSize(imgui.Vec2{screenW, 50})
		imgui.SetNextWindowPos(imgui.Vec2{0, screenH - 50})
		imgui.BeginV("Progress", nil, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove|imgui.WindowFlagsNoTitleBar)
		imgui.ProgressBarV(v.renderingProgress, imgui.Vec2{-1, 0}, fmt.Sprintf("Rendering: %.0f%%", v.renderingProgress*100))
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
			go func() { v.current = v.render() }()
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
