package rules

import (
	"embed"
	"errors"
	"io"
	"os"
	"strings"
)

type RuleConfiguration struct {
	Args   Args         `yaml:"args,omitempty"`
	Script ScriptConfig `yaml:"script"`
	Name   string       `yaml:"name"`
	Users  []string     `yaml:"users,omitempty"`
}

type Args map[string]any

type ScriptConfig struct {
	Packaged string `yaml:"packaged,omitempty"`
	Text     string `yaml:"text,omitempty"`
	Path     string `yaml:"path,omitempty"`
}

func (s ScriptConfig) script(embedFS *embed.FS) (io.ReadCloser, error) {
	switch {
	case s.Text != "":
		return io.NopCloser(strings.NewReader(s.Text)), nil
	case s.Packaged != "":
		return embedFS.Open(s.Packaged)
	case s.Path != "":
		return os.Open(s.Path)
	default:
		return nil, errors.New("script config is empty")
	}
}
