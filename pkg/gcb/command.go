package gcb

import (
	"fmt"
	"sort"
	"strings"
)

type Command struct {
	Code        GCode
	Args        Args
	LineComment string
}

func (c *Command) String(line, above bool) string {
	result := string(c.Code)
	names := make([]string, 0, len(c.Args))
	for name := range c.Args {
		names = append(names, name)
	}

	// sort names
	sort.Strings(names)

	for _, name := range names {
		result += fmt.Sprintf(" %v%f", name, c.Args[name])
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

type Args map[string]RelativePos
