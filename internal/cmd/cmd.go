package cmd

import (
	"codeberg.org/clambin/go-common/charmer"
	"github.com/clambin/tado-exporter/internal/cmd/eval"
	"github.com/clambin/tado-exporter/internal/cmd/monitor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"time"
)

var (
	configFilename string
	RootCmd        = cobra.Command{
		Use:   "tado",
		Short: "Utility for Tadoº thermostats",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			charmer.SetJSONLogger(cmd, viper.GetBool("debug"))
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&configFilename, "config", "", "Configuration file")
	RootCmd.PersistentFlags().Bool("debug", false, "Log debug messages")
	_ = viper.BindPFlag("debug", RootCmd.PersistentFlags().Lookup("debug"))

	RootCmd.AddCommand(&monitor.Cmd, &eval.Cmd)
}

var args = charmer.Arguments{
	"debug":           charmer.Argument{Default: false, Help: "Log debug messages"},
	"pprof":           charmer.Argument{Default: "", Help: "Enable pprof"},
	"tado.username":   charmer.Argument{Default: "", Help: "Tadoº username"},
	"tado.password":   charmer.Argument{Default: "", Help: "Tadoº password"},
	"exporter.addr":   charmer.Argument{Default: ":9090", Help: "Address of Prometheus exporter"},
	"poller.interval": charmer.Argument{Default: 30 * time.Second, Help: "Poller interval"},
	"health.addr":     charmer.Argument{Default: ":8080", Help: "Address of /health endpoint"},
	"slack.token":     charmer.Argument{Default: "", Help: "Slack token"},
}

func initConfig() {
	if configFilename != "" {
		viper.SetConfigFile(configFilename)
	} else {
		viper.AddConfigPath("/etc/tado-monitor/")
		viper.AddConfigPath("$HOME/.tado-monitor")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	if err := charmer.SetDefaults(viper.GetViper(), args); err != nil {
		panic("failed to set viper defaults: " + err.Error())
	}

	viper.SetEnvPrefix("TADO_MONITOR")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Error("failed to read config file", "err", err)
		os.Exit(1)
	}
}
