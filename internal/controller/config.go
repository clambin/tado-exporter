package controller

type Configuration struct {
	HomeRules []RuleConfiguration            `yaml:"homeRules"`
	ZoneRules map[string][]RuleConfiguration `yaml:"zoneRules"`
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
