package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpango/glg"

	pkg "github.com/gucio321/spiffy/pkg"
	"github.com/gucio321/spiffy/pkg/gcb"
	"github.com/gucio321/spiffy/pkg/viewer"
)

type flags struct {
	inputFilePath  string
	outputFilePath string
	scale          float64
	commentsAbove  bool
	noLineComments bool
	view           bool
	repeatN        int
	repeatDepth    float64
}

func main() {
	var f flags
	flag.StringVar(&f.inputFilePath, "i", "", "input file path")
	flag.StringVar(&f.outputFilePath, "o", "", "output file path")
	flag.Float64Var(&f.scale, "s", 1.0, "scale factor")
	flag.BoolVar(&f.noLineComments, "nlc", false, "no line comments")
	flag.BoolVar(&f.commentsAbove, "ca", false, "comments above")
	flag.BoolVar(&f.view, "v", false, "view")
	flag.IntVar(&f.repeatN, "rn", 0, "repeat N times (use with -rd)")
	flag.Float64Var(&f.repeatDepth, "rd", 5, "repeat depth (use with -rn)")
	flag.Parse()

	if _, err := os.Stat(f.inputFilePath); os.IsNotExist(err) {
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(f.inputFilePath)
	if err != nil {
		glg.Fatalf("Cannot read file %s: %v", f.inputFilePath, err)
	}

	result, err := pkg.Parse(data)
	if err != nil {
		glg.Fatalf("Cannot parse file %s: %v", f.inputFilePath, err)
	}

	if f.repeatN > 0 {
		result.Repeat(f.repeatN, float32(f.repeatDepth))
	}

	result.Scale(float32(f.scale))

	gcode := gcb.NewGCodeBuilder()

	gcode.SetDepth(2)

	if err := result.GCode(gcode); err != nil {
		gcode.Dump()
		glg.Fatalf("Cannot generate GCode: %v", err)
	}

	gcode.Comments(!f.noLineComments, f.commentsAbove)

	cmds := gcode.Commands()
	gcode.CleanCommands()

	gcode.PushCommand(gcb.Command{
		Code: gcb.G0,
		Args: map[string]gcb.RelativePos{
			"Z": -19,
		},
	})

	gcode.PushCommand(cmds...)

	fmt.Println(gcode)

	if f.outputFilePath != "" {
		if err := os.WriteFile(f.outputFilePath, []byte(gcode.String()), 0644); err != nil {
			glg.Fatalf("Cannot write file %s: %v", f.outputFilePath, err)
		}
	}

	if f.view {
		ebiten.SetWindowSize(800, 600)
		if err := ebiten.RunGame(viewer.NewViewer(gcode)); err != nil {
			glg.Fatalf("Cannot run viewer: %v", err)
		}
	}
}
