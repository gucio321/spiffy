package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kpango/glg"

	pkg "github.com/gucio321/spiffy/pkg"
)

func main() {
	var inputFilePath string
	flag.StringVar(&inputFilePath, "i", "", "input file path")
	flag.Parse()

	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(inputFilePath)
	if err != nil {
		glg.Fatalf("Cannot read file %s: %v", inputFilePath, err)
	}

	result, err := pkg.Parse(data)
	if err != nil {
		glg.Fatalf("Cannot parse file %s: %v", inputFilePath, err)
	}

	result.Scale(0.1)
	gcode, err := result.GCode()
	if err != nil {
		glg.Fatalf("Cannot generate GCode: %v", err)
	}

	fmt.Println(gcode)
}
