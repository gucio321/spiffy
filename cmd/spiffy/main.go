package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kpango/glg"

	pkg "github.com/gucio321/spiffy/pkg"
)

type flags struct {
	inputFilePath  string
	outputFilePath string
	scale          float64
}

func main() {
	var f flags
	flag.StringVar(&f.inputFilePath, "i", "", "input file path")
	flag.StringVar(&f.outputFilePath, "o", "", "output file path")
	flag.Float64Var(&f.scale, "s", 1.0, "scale factor")
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

	result.Scale(float32(f.scale))
	gcode, err := result.GCode()
	if err != nil {
		glg.Fatalf("Cannot generate GCode: %v", err)
	}

	fmt.Println(gcode)

	if f.outputFilePath != "" {
		if err := os.WriteFile(f.outputFilePath, []byte(gcode.String()), 0644); err != nil {
			glg.Fatalf("Cannot write file %s: %v", f.outputFilePath, err)
		}
	}
}
