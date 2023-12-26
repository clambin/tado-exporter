package poller

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/pkg/pubsub"
	"log/slog"
	"time"
)

type Poller interface {
	Subscribe() chan Update
	Unsubscribe(ch chan Update)
	Refresh()
}

type TadoGetter interface {
	GetWeatherInfo(context.Context) (tado.WeatherInfo, error)
	GetMobileDevices(context.Context) ([]tado.MobileDevice, error)
	GetZones(context.Context) (tado.Zones, error)
	GetZoneInfo(context.Context, int) (tado.ZoneInfo, error)
	GetHomeState(ctx context.Context) (homeState tado.HomeState, err error)
}

var _ Poller = &TadoPoller{}

type TadoPoller struct {
	TadoClient TadoGetter
	*pubsub.Publisher[Update]
	interval time.Duration
	logger   *slog.Logger
	refresh  chan struct{}
}

func New(tadoClient TadoGetter, interval time.Duration, logger *slog.Logger) *TadoPoller {
	return &TadoPoller{
		TadoClient: tadoClient,
		Publisher:  pubsub.New[Update](logger.With(slog.String("component", "registry"))),
		interval:   interval,
		logger:     logger,
		refresh:    make(chan struct{}),
	}
}

func (p *TadoPoller) Run(ctx context.Context) error {
	p.logger.Debug("started", slog.Duration("interval", p.interval))
	defer p.logger.Debug("stopped")

	timer := time.NewTicker(p.interval)
	defer timer.Stop()

	for {
		shouldPoll := false
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			shouldPoll = true
		case <-p.refresh:
			shouldPoll = true
		}

		if shouldPoll {
			// poll for new data
			if err := p.poll(ctx); err != nil {
				p.logger.Error("failed to get tado metrics", slog.Any("err", err))
			}
		}
	}
}

func (p *TadoPoller) Refresh() {
	p.refresh <- struct{}{}
}

func (p *TadoPoller) poll(ctx context.Context) error {
	start := time.Now()
	update, err := p.update(ctx)
	if err == nil {
		p.Publisher.Publish(update)
		p.logger.Debug("poll completed", slog.Duration("duration", time.Since(start)))
	}
	return err
}

func (p *TadoPoller) update(ctx context.Context) (update Update, err error) {
	update.UserInfo, err = p.getMobileDevices(ctx)
	if err == nil {
		update.WeatherInfo, err = p.TadoClient.GetWeatherInfo(ctx)
	}
	if err == nil {
		update.Zones, err = p.getZones(ctx)
	}
	if err == nil {
		update.ZoneInfo, err = p.getZoneInfos(ctx, update.Zones)
	}
	if err == nil {
		update.Home, err = p.getHomeState(ctx)
	}
	return update, err
}

func (p *TadoPoller) getMobileDevices(ctx context.Context) (map[int]tado.MobileDevice, error) {
	var deviceMap map[int]tado.MobileDevice
	devices, err := p.TadoClient.GetMobileDevices(ctx)
	if err == nil {
		deviceMap = make(map[int]tado.MobileDevice)
		for _, device := range devices {
			if device.Settings.GeoTrackingEnabled {
				deviceMap[device.ID] = device
			}
		}
	}
	return deviceMap, err
}

func (p *TadoPoller) getZones(ctx context.Context) (map[int]tado.Zone, error) {
	var zoneMap map[int]tado.Zone
	zones, err := p.TadoClient.GetZones(ctx)
	if err == nil {
		zoneMap = make(map[int]tado.Zone)
		for _, zone := range zones {
			zoneMap[zone.ID] = zone
		}
	}
	return zoneMap, err
}

func (p *TadoPoller) getZoneInfos(ctx context.Context, zones map[int]tado.Zone) (map[int]tado.ZoneInfo, error) {
	zoneInfoMap := make(map[int]tado.ZoneInfo)
	for zoneID := range zones {
		zoneInfo, err := p.TadoClient.GetZoneInfo(ctx, zoneID)
		if err != nil {
			return nil, err
		}

		zoneInfoMap[zoneID] = zoneInfo
	}
	return zoneInfoMap, nil
}

func (p *TadoPoller) getHomeState(ctx context.Context) (IsHome, error) {
	var home IsHome
	homeState, err := p.TadoClient.GetHomeState(ctx)
	if err == nil {
		home = homeState.Presence == "HOME"
	}
	return home, err
}
