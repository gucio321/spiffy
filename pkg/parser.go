package spiffy

import (
	"encoding/xml"
)

func Parse(data []byte) (result *Spiffy, err error) {
	// 0.0: initialize
	result = NewSpiffy()

	// 1.0: unmarshal xml
	if err := xml.Unmarshal(data, result); err != nil {
		return nil, err
	}

	// N.N: return
	return result, nil
}
