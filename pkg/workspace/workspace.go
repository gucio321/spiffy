package workspace

import (
	_ "embed"
	"encoding/json"
)

//go:embed workspaces.json
var workspaces []byte

// Workspace represents our working area.
type Workspace struct {
	// MinX and MinY represent the point counting from printers (0,0)
	MinX, MinY,
	// MaxX and MaxY represent the point counting from printers (0,0)
	MaxX, MaxY int

	Name        string
	Description string
}

func decodeWorkspaces() ([]Workspace, error) {
	var result []Workspace
	if err := json.Unmarshal(workspaces, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func Get(name string) (*Workspace, error) {
	workspaces, err := decodeWorkspaces()
	if err != nil {
		return nil, err
	}

	for _, workspace := range workspaces {
		if workspace.Name == name {
			return &workspace, nil
		}
	}

	return nil, nil
}
