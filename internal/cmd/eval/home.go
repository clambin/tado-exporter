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

var homeCmd = cobra.Command{
	Use:   "home",
	Short: "Evaluate a Lua home rule script",
	RunE:  evalHomeRule(os.Stdout, viper.GetViper()),
}

func evalHomeRule(w io.Writer, v *viper.Viper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			args = []string{"-"}
		}
		for _, arg := range args {
			r, err := evalHomeRuleScript(arg, v.GetBool("action-only"))
			if err != nil {
				return err
			}
			r.writeTo(w)
		}
		return nil
	}
}

func evalHomeRuleScript(path string, actionOnly bool) (r results, err error) {
	var rule rules.Rule
	if rule, err = loadHomeRule(path); err == nil {
		r, err = evalRule(rule, actionOnly, homeRuleInput())
	}
	return r, err
}

func loadHomeRule(path string) (r rules.Rule, err error) {
	var cfg rules.RuleConfiguration
	if cfg, err = getRuleConfig(path); err == nil {
		r, err = rules.LoadHomeRule(cfg)
	}
	return r, err
}

func homeRuleInput() iter.Seq2[rules.State, string] {
	return func(yield func(rules.State, string) bool) {
		for _, homeOverlay := range []bool{false, true} {
			for _, homeMode := range []bool{false, true} {
				homeDesc := fmt.Sprintf("home(overlay:%v,home:%v) ", homeOverlay, homeMode)
				for _, userHome := range []bool{false, true} {
					desc := homeDesc + fmt.Sprintf("user(home:%v)", userHome)
					s := rules.State{
						HomeId:    1,
						HomeState: rules.HomeState{Overlay: homeOverlay, Home: homeMode},
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
