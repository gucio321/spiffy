package gcb

import (
	"strconv"
	"strings"

	"github.com/gucio321/spiffy/pkg/workspace"
	"github.com/kpango/glg"
)

func NewGCodeBuilderFromGCode(gcode []byte) (*GCodeBuilder, error) {
	workspace, err := workspace.Get(DefaultWorkspace)
	if err != nil {
		return nil, err
	}

	result := NewGCodeBuilder(workspace)
	lines := strings.Split(string(gcode), "\n")
	positioning := G90

	for _, line := range lines {
		splitted := strings.Split(line, ";")
		command := splitted[0]
		comment := strings.Join(splitted[1:], ";")

		// trim unnecessary spaces from command
		command = strings.TrimSpace(command)
		sommandParts := strings.Split(command, " ")
		code := GCode(sommandParts[0])

		switch code {
		case G90:
			positioning = G90
		case G91:
			positioning = G91
		default:
			if positioning == G90 {
				glg.Warnf("Got \"%s\" command but is in Relative Positioning mode which is not supported", code)
				continue
			}
		}

		args := make(map[string]RelativePos)
		for _, arg := range sommandParts[1:] {
			if len(arg) <= 1 {
				continue
			}

			value, err := strconv.ParseFloat(arg[1:], 32)
			if err != nil {
				return nil, err
			}

			args[arg[0:1]] = RelativePos(value)
		}

		newCommand := Command{
			Code:        code,
			LineComment: comment,
			Args:        args,
		}

		result.PushCommand(newCommand)
	}

	return result, nil
}
