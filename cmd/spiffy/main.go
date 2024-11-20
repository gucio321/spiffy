package main

import (
	"flag"
	"fmt"
	"os"

	inkscape "github.com/galihrivanto/go-inkscape"
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
	startZ         float64
	depthDelta     float64
	force          bool
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
	flag.Float64Var(&f.startZ, "sz", 0, "start Z (use along with -dz for delta zet)")
	flag.Float64Var(&f.depthDelta, "dz", gcb.BaseDepth, "delta Z (use along with -sz for start zet)")
	flag.BoolVar(&f.force, "f", false, "force")
	flag.Parse()

	if f.startZ != 0 && f.depthDelta == gcb.BaseDepth && !f.force {
		glg.Fatal("Please specify -dz (-f to force)")
	}

	if _, err := os.Stat(f.inputFilePath); os.IsNotExist(err) {
		flag.Usage()
		os.Exit(1)
	}

	inkscapeProxy := inkscape.NewProxy(inkscape.Verbose(false))
	if err := inkscapeProxy.Run(); err != nil {
		glg.Fatalf("Cannot run inkscape: %v", err)
	}

	defer inkscapeProxy.Close()

	convertedFile := f.inputFilePath + ".spiffy.svg"
	inkscapeProxy.RawCommands(
		fmt.Sprintf("file-open:%s", f.inputFilePath),
		fmt.Sprintf("export-filename:%s", convertedFile),
		"export-type:svg",
		"select-all",
		"object-to-path",
		"export-do",
	)

	data, err := os.ReadFile(convertedFile)
	if err != nil {
		glg.Fatalf("Cannot read file %s: %v", f.inputFilePath, err)
	}

	result, err := pkg.Parse(data)
	if err != nil {
		glg.Fatalf("Cannot parse file %s: %v", f.inputFilePath, err)
	}

	if f.repeatN > 0 {
		result.Repeat(f.repeatN, f.repeatDepth)
	}

	if f.startZ != 0 {
		result.Depths(f.depthDelta, f.startZ)
	}

	result.Scale(float32(f.scale))
	gcode, err := result.GCode()
	if err != nil {
		gcode.Dump()
		glg.Fatalf("Cannot generate GCode: %v", err)
	}

	gcode.Comments(!f.noLineComments, f.commentsAbove)

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
