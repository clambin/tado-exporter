package main

import (
	"github.com/clambin/tado-exporter/cmd"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
)

var (
	// overridden during build
	version = "change-me"
)

func main() {
	go func() { _ = http.ListenAndServe(":6000", nil) }()

	cmd.RootCmd.Version = version
	if err := cmd.RootCmd.Execute(); err != nil {
		slog.Error("failed to start", "err", err)
		os.Exit(1)
	}
}
