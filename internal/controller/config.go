package controller

type Configuration struct {
	Zones map[string][]RuleConfiguration `yaml:"zones"`
	Home  []RuleConfiguration            `yaml:"home"`
}

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
