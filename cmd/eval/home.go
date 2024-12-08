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
			if err := evalHomeRuleScript(w, arg, v.GetBool("action-only")); err != nil {
				return err
			}
		}
		return nil
	}
}

func evalHomeRuleScript(w io.Writer, path string, actionOnly bool) error {
	r, err := getHomeScript(path)
	if err != nil {
		return err
	}

	var headerPrinted bool
	for s, description := range homeRuleInput() {
		a, err := r.Evaluate(s)
		if err != nil {
			return err
		}
		if actionOnly && a.IsState(s) {
			continue
		}

		const formatString = "%-90s %-6v %-40s %s\n"

		if !headerPrinted {
			_, _ = fmt.Fprintf(w, formatString, "INPUT", "CHANGE", "REASON", "ACTION")
			headerPrinted = true
		}

		_, _ = fmt.Fprintf(w, formatString, description, !a.IsState(s), a.Reason(), a.Description(true))
	}
	return nil
}

func getHomeScript(filename string) (rules.Rule, error) {
	var cfg rules.RuleConfiguration
	if filename == "-" {
		s, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		cfg = rules.RuleConfiguration{Script: rules.ScriptConfig{Text: string(s)}}
	} else {
		cfg = rules.RuleConfiguration{Script: rules.ScriptConfig{Path: filename}}
	}
	return rules.LoadHomeRule(cfg)
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
