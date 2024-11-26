package controller

import (
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

func TestRuleConfiguration(t *testing.T) {
	cfg := Configuration{
		HomeRules: []RuleConfiguration{
			{
				Name: "auto away",
				Script: ScriptConfig{Text: `
function Evaluate(state, devices)
	if #devices == 0 then
	 	return state, 0, "no devices found"
	end
	homeUsers = getDevicesByState(devices, true)
	if #homeUsers == 0 then
		return "away", 300, "all users are away"
    end
	return "home", 0, "one or more users are home"
end

function getDevicesByState(list, state)
    local result = {}
    for _, obj in ipairs(list) do
        if obj.Home == state then
			table.insert(result, obj)
        end
    end
    return result
end
`,
				},
			},
		},
		ZoneRules: map[string][]RuleConfiguration{
			"Bathroom": {
				{
					Name:   "LimitOverlay",
					Script: ScriptConfig{Path: "/path/script.lua"},
				},
				{
					Name:   "nighttime",
					Script: ScriptConfig{Packaged: "nighttime"},
				},
			},
		},
	}
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		t.Fatal(err)
	}
}
