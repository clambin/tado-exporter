package config

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
)

type Encoder interface {
	Encode(any) error
}

type TadoGetter interface {
	GetZones(context.Context) (tado.Zones, error)
	GetMobileDevices(context.Context) ([]tado.MobileDevice, error)
}

type entry struct {
	ID   int
	Name string
}

type report struct {
	Zones   []entry
	Devices []entry
}

func ShowConfig(ctx context.Context, c TadoGetter, e Encoder) error {
	var r report

	zones, err := c.GetZones(ctx)
	if err != nil {
		return fmt.Errorf("tado: zones: %w", err)
	}
	for _, zone := range zones {
		r.Zones = append(r.Zones, entry{
			ID:   zone.ID,
			Name: zone.Name,
		})
	}

	devices, err := c.GetMobileDevices(ctx)
	if err != nil {
		return fmt.Errorf("tado: mobileDevices: %w", err)
	}
	for _, device := range devices {
		r.Devices = append(r.Devices, entry{
			ID:   device.ID,
			Name: device.Name,
		})
	}

	return e.Encode(r)
}
