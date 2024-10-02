package gcb

import "errors"

var (
	ErrCantChangeDrawingState           = errors.New("cannot change drawing state")
	ErrInvalidContinousLineContinuation = errors.New("invalid continous line continuation - current position does not match estimated start position.")
)
