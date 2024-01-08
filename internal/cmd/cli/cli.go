package cli

import (
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
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&configFilename, "config", "", "Configuration file")
	RootCmd.PersistentFlags().Bool("debug", false, "Log debug messages")
	_ = viper.BindPFlag("debug", RootCmd.PersistentFlags().Lookup("debug"))
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

	viper.SetDefault("debug", false)
	viper.SetDefault("tado.username", "")
	viper.SetDefault("tado.password", "")
	viper.SetDefault("tado.clientSecret", "")
	viper.SetDefault("exporter.addr", ":9090")
	viper.SetDefault("poller.interval", 30*time.Second)
	viper.SetDefault("health.addr", ":8080")
	viper.SetDefault("controller.tadobot.enabled", true)
	viper.SetDefault("controller.tadobot.token", "")

	viper.SetEnvPrefix("TADO_MONITOR")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Error("failed to read config file", "err", err)
		os.Exit(1)
	}
}
