package config

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"os"
)

var (
	Cmd = cobra.Command{
		Use:   "config",
		Short: "Show Tado configuration",
		RunE:  showConfig,
	}
)

func showConfig(cmd *cobra.Command, _ []string) error {
	api, err := tado.New(
		viper.GetString("tado.username"),
		viper.GetString("tado.password"),
		viper.GetString("tado.clientSecret"),
	)
	if err != nil {
		return fmt.Errorf("tado: %w", err)
	}

	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)

	return ShowConfig(cmd.Context(), api, enc)
}

type Encoder interface {
	Encode(any) error
}

type TadoGetter interface {
	GetZones(context.Context) (tado.Zones, error)
	GetMobileDevices(context.Context) ([]tado.MobileDevice, error)
}

type report struct {
	Zones   []map[string]any `json:"zones"`
	Devices []map[string]any `json:"devices"`
}

func ShowConfig(ctx context.Context, c TadoGetter, e Encoder) error {
	var r report

	zones, err := c.GetZones(ctx)
	if err != nil {
		return fmt.Errorf("tado: zones: %w", err)
	}
	for _, zone := range zones {
		r.Zones = append(r.Zones, map[string]any{
			"id":   zone.ID,
			"name": zone.Name,
		})
	}

	devices, err := c.GetMobileDevices(ctx)
	if err != nil {
		return fmt.Errorf("tado: mobileDevices: %w", err)
	}
	for _, device := range devices {
		r.Devices = append(r.Devices, map[string]any{
			"id":       device.ID,
			"name":     device.Name,
			"tracking": device.Settings.GeoTrackingEnabled,
		})
	}

	return e.Encode(r)
}
