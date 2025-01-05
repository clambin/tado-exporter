package eval

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"iter"
	"os"
)

var zoneCmd = cobra.Command{
	Use:   "zone",
	Short: "Evaluate a Lua zone rule script",
	RunE:  evalZoneRule(os.Stdout, viper.GetViper()),
}

func evalZoneRule(w io.Writer, v *viper.Viper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			args = []string{"-"}
		}
		for _, arg := range args {
			r, err := evalZoneRuleScript(arg, v.GetBool("action-only"))
			if err != nil {
				return err
			}
			r.writeTo(w)
		}
		return nil
	}
}

func evalZoneRuleScript(path string, actionOnly bool) (r results, err error) {
	var rule rules.Rule
	if rule, err = loadZoneRule(path); err == nil {
		r, err = evalRule(rule, actionOnly, zoneRuleInput())
	}
	return r, err
}

func loadZoneRule(filename string) (r rules.Rule, err error) {
	var cfg rules.RuleConfiguration
	if cfg, err = getRuleConfig(filename); err == nil {
		r, err = rules.LoadZoneRule("zone", cfg)
	}
	return r, err
}

func zoneRuleInput() iter.Seq2[rules.State, string] {
	return func(yield func(rules.State, string) bool) {
		for _, homeOverlay := range []bool{false, true} {
			for _, homeMode := range []bool{false, true} {
				homeDesc := fmt.Sprintf("home(overlay:%v,home:%v) ", homeOverlay, homeMode)
				for _, zoneOverlay := range []bool{false, true} {
					for _, zoneHeating := range []bool{false, true} {
						zoneDesc := homeDesc + fmt.Sprintf("zone(overlay:%v,heating: %v) ", zoneOverlay, zoneHeating)
						for _, userHome := range []bool{false, true} {
							desc := zoneDesc + fmt.Sprintf("user(home:%v)", userHome)
							s := rules.State{
								HomeId:    1,
								ZoneId:    10,
								HomeState: rules.HomeState{Overlay: homeOverlay, Home: homeMode},
								ZoneState: rules.ZoneState{Overlay: zoneOverlay, Heating: zoneHeating},
								Devices:   rules.Devices{rules.Device{Name: "user", Home: userHome}},
							}
							if !yield(s, desc) {
								return
							}
						}
					}
				}
			}
		}
	}
}
