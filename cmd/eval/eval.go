package eval

import (
	"github.com/clambin/go-common/charmer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	Cmd = cobra.Command{
		Use:   "eval",
		Short: "evaluate a Lua rules script",
	}
)

var args = charmer.Arguments{
	"action-only": {
		Default: false,
		Help:    "only print states that results in an action",
	},
}

func init() {
	_ = charmer.SetPersistentFlags(&Cmd, viper.GetViper(), args)
	Cmd.AddCommand(&zoneCmd, &homeCmd)
}
