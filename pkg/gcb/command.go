package gcb

import (
	"fmt"
	"strings"
)

type Command struct {
	Code        string
	Args        []Arg
	LineComment string
}

func (c *Command) String(line, above bool) string {
	result := c.Code
	for _, arg := range c.Args {
		result += fmt.Sprintf(" %v%f", arg.Name, arg.Value)
	}

	if c.LineComment != "" {
		switch {
		case (line && c.Code != "") || (above && c.Code == ""):
			result += fmt.Sprintf(" ; %v", c.LineComment)
		}
	}

	// merge duplicated spaces
	for {
		old := result
		result = strings.ReplaceAll(result, "  ", " ")
		result = strings.ReplaceAll(result, "\n\n", "\n")
		if old == result {
			break
		}
	}

	return result
}

type Arg struct {
	Name  string
	Value RelativePos
}
