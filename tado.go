package main

import (
	"github.com/clambin/tado-exporter/internal/cmd"
	"log/slog"
	"os"
)

var (
	// overridden during build
	version = "change-me"
)

func main() {
	cmd.RootCmd.Version = version
	if err := cmd.RootCmd.Execute(); err != nil {
		slog.Error("failed to start", "err", err)
		os.Exit(1)
	}
}
