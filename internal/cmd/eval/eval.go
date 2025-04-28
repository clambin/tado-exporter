package eval

import (
	"codeberg.org/clambin/go-common/charmer"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"iter"
	"os"
)

var (
	Cmd = cobra.Command{
		Use:   "eval",
		Short: "evaluate a Lua rules script",
	}

	args = charmer.Arguments{
		"action-only": {Default: false, Help: "only print states that results in an action"},
	}
)

func init() {
	_ = charmer.SetPersistentFlags(&Cmd, viper.GetViper(), args)
	Cmd.AddCommand(&zoneCmd, &homeCmd)
}

func getRuleConfig(filename string) (rules.RuleConfiguration, error) {
	var r io.ReadCloser
	var err error
	switch filename {
	case "-":
		r = os.Stdin
	default:
		r, err = os.Open(filename)
		if err != nil {
			return rules.RuleConfiguration{}, err
		}
		defer func() { _ = r.Close() }()
	}
	var body []byte
	if body, err = io.ReadAll(r); err != nil {
		return rules.RuleConfiguration{}, err
	}
	return rules.RuleConfiguration{Script: rules.ScriptConfig{Text: string(body)}}, nil
}

const formatString = "%-90s %-6v %-40s %s\n"

type results []result

func evalRule(rule rules.Rule, actionOnly bool, input iter.Seq2[rules.State, string]) (results, error) {
	var r results
	for s, description := range input {
		a, err := rule.Evaluate(s)
		if err != nil {
			return nil, err
		}
		if a.IsState(s) && actionOnly {
			continue
		}
		r = append(r, result{action: a, description: description, change: !a.IsState(s)})
	}
	return r, nil
}

func (r results) writeTo(w io.Writer) {
	if len(r) > 0 {
		_, _ = fmt.Fprintf(w, formatString, "INPUT", "CHANGE", "REASON", "ACTION")
		for _, res := range r {
			res.writeTo(w)
		}
	}
}

type result struct {
	action      rules.Action
	description string
	change      bool
}

func (r result) writeTo(w io.Writer) {
	_, _ = fmt.Fprintf(w, formatString, r.description, r.change, r.action.Reason(), r.action.Description(true))
}
