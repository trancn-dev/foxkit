package manifest

import (
	_ "embed"
	"encoding/json"
)

//go:embed commands.json
var Raw []byte

type Param struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Required  bool     `json:"required,omitempty"`
	Default   any      `json:"default,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Options   []string `json:"options,omitempty"`
	Multiple  bool     `json:"multiple,omitempty"`
	AllowDir  bool     `json:"allowDir,omitempty"`
	Separator string   `json:"separator,omitempty"`
	Label     string   `json:"label,omitempty"`
	Confirm   string   `json:"confirm,omitempty"`
}

type Example struct {
	Cmd  string `json:"cmd"`
	Desc string `json:"desc"`
}

type Command struct {
	Name            string    `json:"name"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Category        string    `json:"category"`
	Args            []Param   `json:"args"`
	Flags           []Param   `json:"flags"`
	Admin           bool      `json:"admin"`
	Confirm         string    `json:"confirm,omitempty"`
	LongRunning     bool      `json:"longRunning"`
	LongRunningFlag string    `json:"longRunningFlag,omitempty"`
	DryRun          bool      `json:"dryRun"`
	Output          string    `json:"output"`
	Examples        []Example `json:"examples,omitempty"`
}

type File struct {
	Schema   int       `json:"schema"`
	Commands []Command `json:"commands"`
}

// Parse giải mã commands.json nhúng sẵn.
func Parse() (File, error) {
	var f File
	err := json.Unmarshal(Raw, &f)
	return f, err
}
