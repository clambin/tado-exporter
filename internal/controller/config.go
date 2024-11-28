package controller

type Configuration struct {
	Home  []RuleConfiguration            `yaml:"home"`
	Zones map[string][]RuleConfiguration `yaml:"zones"`
}

type RuleConfiguration struct {
	Name   string       `yaml:"name"`
	Script ScriptConfig `yaml:"script"`
	Users  []string     `yaml:"users,omitempty"`
}

type ScriptConfig struct {
	Packaged string `yaml:"packaged,omitempty"`
	Text     string `yaml:"text,omitempty"`
	Path     string `yaml:"path,omitempty"`
}
