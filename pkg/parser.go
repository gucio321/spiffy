package spiffy

import "github.com/rustyoz/svg"

func Parse(data []byte) (result *Spiffy, err error) {
	// 0.0: initialize
	result = NewSpiffy()

	// 1.0: unmarshal xml
	// TODO: 2nd arg is "name" (what the hell it means?) and 3rd i "scale" (probably could be 1
	if result.svg, err = svg.ParseSvg(string(data), "", 1); err != nil {
		return nil, err
	}

	// N.N: return
	return result, nil
}
