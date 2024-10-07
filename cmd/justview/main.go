package main

import (
	"flag"
	"os"

	"github.com/gucio321/spiffy/pkg/gcb"
	"github.com/gucio321/spiffy/pkg/viewer"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kpango/glg"
)

func main() {
	inputFile := flag.String("i", "", "Input file")
	flag.Parse()

	if *inputFile == "" {
		flag.Usage()
		glg.Fatal("Input file is required")
	}

	// load file
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		glg.Fatal(err)
	}

	// parse file
	builder, err := gcb.NewGCodeBuilderFromGCode(data)
	if err != nil {
		glg.Fatal(err)
	}

	if err := ebiten.RunGame(viewer.NewViewer(builder)); err != nil {
		glg.Fatal(err)
	}
}
