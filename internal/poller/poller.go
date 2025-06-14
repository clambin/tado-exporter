package poller

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"codeberg.org/clambin/go-common/pubsub"
	"github.com/clambin/tado/v2"
	"github.com/clambin/tado/v2/tools"
)

type TadoClient interface {
	GetMeWithResponse(ctx context.Context, reqEditors ...tado.RequestEditorFn) (*tado.GetMeResponse, error)
	GetZonesWithResponse(ctx context.Context, homeId tado.HomeId, reqEditors ...tado.RequestEditorFn) (*tado.GetZonesResponse, error)
	GetZoneStateWithResponse(ctx context.Context, homeId tado.HomeId, zoneId tado.ZoneId, reqEditors ...tado.RequestEditorFn) (*tado.GetZoneStateResponse, error)
	GetMobileDevicesWithResponse(ctx context.Context, homeId tado.HomeId, reqEditors ...tado.RequestEditorFn) (*tado.GetMobileDevicesResponse, error)
	GetWeatherWithResponse(ctx context.Context, homeId tado.HomeId, reqEditors ...tado.RequestEditorFn) (*tado.GetWeatherResponse, error)
	GetHomeStateWithResponse(ctx context.Context, homeId tado.HomeId, reqEditors ...tado.RequestEditorFn) (*tado.GetHomeStateResponse, error)
	//tado.ClientWithResponsesInterface
}

type Poller struct {
	TadoClient
	logger  *slog.Logger
	refresh chan struct{}
	pubsub.Publisher[Update]
	interval time.Duration
	HomeId   tado.HomeId
}

func New(tadoClient TadoClient, interval time.Duration, logger *slog.Logger) *Poller {
	return &Poller{
		TadoClient: tadoClient,
		Publisher:  pubsub.Publisher[Update]{},
		interval:   interval,
		logger:     logger,
		refresh:    make(chan struct{}),
	}
}

func (p *Poller) Run(ctx context.Context) error {
	p.logger.Debug("started", slog.Duration("interval", p.interval))
	defer p.logger.Debug("stopped")

	timer := time.NewTicker(p.interval)
	defer timer.Stop()

	for {
		if err := p.poll(ctx); err != nil {
			p.logger.Error("failed to get tado metrics", slog.Any("err", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		case <-p.refresh:
		}
	}
}

func (p *Poller) Refresh() {
	p.refresh <- struct{}{}
}

func (p *Poller) poll(ctx context.Context) error {
	//start := time.Now()
	update, err := p.update(ctx)
	if err == nil {
		p.Publish(update)
		//p.logger.Debug("poll completed", slog.Duration("duration", time.Since(start)))
	}
	return err
}

func (p *Poller) update(ctx context.Context) (Update, error) {
	// tools.GetHomes gives detailed tado errors on non-200 responses.
	// For the remaining calls, we'll just report the HTTP status on failure
	// (as they're unlikely to happen if GetHomes succeeded).
	homes, err := tools.GetHomes(ctx, p.TadoClient)
	if err != nil {
		return Update{}, fmt.Errorf("GetHomes: %w", err)
	}
	if len(homes) > 1 {
		return Update{}, fmt.Errorf("only one home supported")
	}

	update := Update{HomeBase: homes[0]}
	homeId := *update.HomeBase.Id
	update.HomeState, err = p.getHomeState(ctx, homeId)
	if err == nil {
		update.Zones, err = p.getZones(ctx, homeId)
	}
	if err == nil {
		update.MobileDevices, err = p.getMobileDevices(ctx, homeId)
	}
	if err == nil {
		update.Weather, err = p.getWeather(ctx, homeId)
	}
	return update, err
}

func (p *Poller) getZones(ctx context.Context, homeId tado.HomeId) ([]Zone, error) {
	zones, err := p.GetZonesWithResponse(ctx, homeId)
	if err != nil {
		return nil, fmt.Errorf("GetZonesWithResponse: %w", err)
	}
	if zones.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GetZonesWithResponse: %d - %s", zones.StatusCode(), zones.Status())
	}
	zoneUpdates := make([]Zone, 0, len(*zones.JSON200))

	for _, zone := range *zones.JSON200 {
		resp, err := p.GetZoneStateWithResponse(ctx, homeId, *zone.Id)
		if err != nil {
			return nil, fmt.Errorf("GetZoneStateWithResponse: %w", err)
		}
		if resp.StatusCode() != http.StatusOK {
			return nil, fmt.Errorf("GetZoneStateWithResponse: %d - %s", resp.StatusCode(), resp.Status())
		}
		zoneUpdates = append(zoneUpdates, Zone{Zone: zone, ZoneState: *resp.JSON200})
	}
	return zoneUpdates, nil
}

func (p *Poller) getMobileDevices(ctx context.Context, homeId tado.HomeId) ([]tado.MobileDevice, error) {
	resp, err := p.GetMobileDevicesWithResponse(ctx, homeId)
	if err != nil {
		return nil, fmt.Errorf("GetMobileDevicesWithResponse: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("GetMobileDevicesWithResponse: %d - %s", resp.StatusCode(), resp.Status())
	}
	return *resp.JSON200, nil
}

func (p *Poller) getWeather(ctx context.Context, homeId tado.HomeId) (tado.Weather, error) {
	resp, err := p.GetWeatherWithResponse(ctx, homeId)
	if err != nil {
		return tado.Weather{}, fmt.Errorf("GetWeatherWithResponse: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return tado.Weather{}, fmt.Errorf("GetWeatherWithResponse: %d - %s", resp.StatusCode(), resp.Status())
	}
	return *resp.JSON200, nil
}

func (p *Poller) getHomeState(ctx context.Context, homeId tado.HomeId) (tado.HomeState, error) {
	resp, err := p.GetHomeStateWithResponse(ctx, homeId)
	if err != nil {
		return tado.HomeState{}, fmt.Errorf("GetHomeStateWithResponse: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return tado.HomeState{}, fmt.Errorf("GetHomeStateWithResponse: %d - %s", resp.StatusCode(), resp.Status())
	}
	return *resp.JSON200, err
}
