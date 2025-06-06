package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	inkscape "github.com/galihrivanto/go-inkscape"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpango/glg"

	pkg "github.com/gucio321/spiffy/pkg"
	"github.com/gucio321/spiffy/pkg/gcb"
	"github.com/gucio321/spiffy/pkg/viewer"
	"github.com/gucio321/spiffy/pkg/workspace"
)

type Flags struct {
	// InputFile represents an SVG file
	InputFilePath string
	// OutputFile is a path to the gcode.
	OutputFilePath string
	// Scale up/down SVG
	Scale float64
	// CommentsAbove puts additional debug comments in gcode if true.
	CommentsAbove bool
	// NoLineComments removes comments with position hints if true
	NoLineComments bool
	// View starts viewer app
	View bool
	// RepeatN - execute svg once go down a bit and repeat N times.
	RepeatN int
	// RepeatDepth - go down this much after each repeat.
	RepeatDepth float64
	// startZ is called calibrationZ in another part of the code.
	// It describes how much to go down before starting code execution.
	StartZ float64
	// DepthDelta describes delta between draw/not draw state.
	DepthDelta float64
	// WorkspaceName is a workspace name from workspaces.json
	WorkspaceName string
	// Workspace is a custom workspace
	Workspace  *workspace.Workspace
	force      bool
	preset     string
	makePreset bool
	showGCode  bool
}

func main() {
	var f Flags
	f.Workspace = &workspace.Workspace{
		Name:        "custom",
		Description: "Custom workspace set by cmd/spiffy",
	}
	flag.StringVar(&f.InputFilePath, "i", "", "input file path")
	flag.StringVar(&f.OutputFilePath, "o", "", "output file path")
	flag.Float64Var(&f.Scale, "s", 1.0, "Scale factor")
	flag.BoolVar(&f.NoLineComments, "nlc", false, "no line comments")
	flag.BoolVar(&f.CommentsAbove, "ca", false, "comments above")
	flag.BoolVar(&f.View, "v", false, "view")
	flag.IntVar(&f.RepeatN, "rn", 0, "repeat N times (use with -rd)")
	flag.Float64Var(&f.RepeatDepth, "rd", 5, "repeat depth (use with -rn)")
	flag.Float64Var(&f.StartZ, "sz", 0, "start Z (use along with -dz for delta zet)")
	flag.Float64Var(&f.DepthDelta, "dz", float64(gcb.BaseDepth), "delta Z (use along with -sz for start zet)")
	flag.BoolVar(&f.force, "f", false, "force")
	flag.StringVar(&f.preset, "preset", "", "JSON preset file path. This will override all other flags")
	flag.BoolVar(&f.makePreset, "make-preset", false, "auto-generate preset")
	flag.BoolVar(&f.showGCode, "show-gcode", false, "print resulting GCode even if -o is set")
	flag.StringVar(&f.WorkspaceName, "workspace", "", "workspace name from workspaces.json")
	flag.IntVar(&f.Workspace.MinX, "minx", 0, "workspace min x")
	flag.IntVar(&f.Workspace.MinY, "miny", 0, "workspace min y")
	flag.IntVar(&f.Workspace.MaxX, "maxx", 0, "workspace max x")
	flag.IntVar(&f.Workspace.MaxY, "maxy", 0, "workspace max y")
	flag.Parse()

	if f.makePreset {
		out, err := json.MarshalIndent(f, "", "\t")
		if err != nil {
			glg.Fatalf("Unable to generate preset: %v", err)
		}
		fmt.Println(string(out))
		glg.Infof("Presets generated")
		return
	}

	if f.preset != "" {
		data, err := os.ReadFile(f.preset)
		if err != nil {
			glg.Fatalf("Unable to read preset from %s: %v (use valid file or empty to not use presets)", f.preset, err)
		}

		if err := json.Unmarshal(data, &f); err != nil {
			glg.Fatalf("Unable to parse preset from %s: %v", f.preset, err)
		}
	}

	if f.StartZ != 0 && f.DepthDelta == float64(gcb.BaseDepth) && !f.force {
		glg.Fatal("Please specify -dz (-f to force)")
	}

	if _, err := os.Stat(f.InputFilePath); os.IsNotExist(err) {
		flag.Usage()
		os.Exit(1)
	}

	inkscapeProxy := inkscape.NewProxy(inkscape.Verbose(true))
	if err := inkscapeProxy.Run(); err != nil {
		glg.Fatalf("Cannot run inkscape: %v", err)
	}

	defer inkscapeProxy.Close()

	glg.Infof("running inkscape pre-processing")
	convertedFile := f.InputFilePath + ".spiffy.svg"
	inkscapeProxy.RawCommands(
		fmt.Sprintf("file-open:%s", f.InputFilePath),
		fmt.Sprintf("export-filename:%s", convertedFile),
		"export-type:svg",
		"select-all",
		"object-to-path",
		"path-simplify",
		"export-do",
	)

	glg.Info("inkscape done.")

	data, err := os.ReadFile(convertedFile)
	if err != nil {
		glg.Fatalf("Cannot read file %s: %v", f.InputFilePath, err)
	}

	result, err := pkg.Parse(data)
	if err != nil {
		glg.Fatalf("Cannot parse file %s: %v", f.InputFilePath, err)
	}

	if f.WorkspaceName != "" {
		result.WorkspaceName(f.WorkspaceName)
	}

	if f.Workspace.MinX != 0 || f.Workspace.MinY != 0 || f.Workspace.MaxX != 0 || f.Workspace.MaxY != 0 {
		result.Workspace(f.Workspace)
	}

	if f.RepeatN > 0 {
		result.Repeat(f.RepeatN, f.RepeatDepth)
	}

	if f.StartZ != 0 {
		result.Depths(f.DepthDelta, f.StartZ)
	}

	result.Scale(float32(f.Scale))
	gcode, err := result.GCode()
	if err != nil {
		gcode.Dump()
		glg.Fatalf("Cannot generate GCode: %v", err)
	}

	gcode.Comments(!f.NoLineComments, f.CommentsAbove)

	if (f.OutputFilePath == "" && !f.View) || f.showGCode {
		fmt.Println(gcode)
	}

	if f.OutputFilePath != "" {
		if err := os.WriteFile(f.OutputFilePath, []byte(gcode.String()), 0644); err != nil {
			glg.Fatalf("Cannot write file %s: %v", f.OutputFilePath, err)
		}
	}

	if f.View {
		ebiten.SetWindowSize(800, 600)
		if err := ebiten.RunGame(viewer.NewViewer(gcode)); err != nil {
			glg.Fatalf("Cannot run viewer: %v", err)
		}
	}
}
