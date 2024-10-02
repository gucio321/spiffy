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

func (c *Command) String(comments bool) string {
	result := c.Code
	for _, arg := range c.Args {
		result += fmt.Sprintf(" %v%v", arg.Name, arg.Value)
	}

	if c.LineComment != "" && comments {
		result += fmt.Sprintf(" ; %v", c.LineComment)
	}

	// merge duplicated spaces
	for {
		old := result
		result = strings.ReplaceAll(result, "  ", " ")
		if old == result {
			break
		}
	}

	return result
}

type Arg struct {
	Name  string
	Value any
}
