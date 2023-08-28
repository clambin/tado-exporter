package poller

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"log/slog"
	"sync"
	"time"
)

type Poller interface {
	Register() chan *Update
	Unregister(ch chan *Update)
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
	API      TadoGetter
	interval time.Duration
	refresh  chan struct{}
	registry map[chan *Update]struct{}
	lock     sync.RWMutex
}

func New(API TadoGetter, interval time.Duration) *TadoPoller {
	return &TadoPoller{
		API:      API,
		interval: interval,
		refresh:  make(chan struct{}),
		registry: make(map[chan *Update]struct{}),
	}
}

func (p *TadoPoller) Run(ctx context.Context) error {
	timer := time.NewTicker(p.interval)
	defer timer.Stop()

	slog.Info("poller started", "interval", p.interval)

	for {
		shouldPoll := false
		select {
		case <-ctx.Done():
			slog.Info("poller stopped")
			return nil
		case <-timer.C:
			shouldPoll = true
		case <-p.refresh:
			shouldPoll = true
		}

		if shouldPoll {
			// poll for new data
			if err := p.poll(ctx); err != nil {
				slog.Error("failed to get tado metrics", "err", err)
			}
		}
	}
}

func (p *TadoPoller) Refresh() {
	p.refresh <- struct{}{}
}

func (p *TadoPoller) Register() chan *Update {
	p.lock.Lock()
	defer p.lock.Unlock()
	ch := make(chan *Update, 1)
	p.registry[ch] = struct{}{}
	slog.Debug(fmt.Sprintf("poller has %d clients", len(p.registry)))
	return ch
}

func (p *TadoPoller) Unregister(ch chan *Update) {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.registry, ch)
	slog.Debug(fmt.Sprintf("poller has %d clients", len(p.registry)))
}

func (p *TadoPoller) poll(ctx context.Context) error {
	start := time.Now()
	update, err := p.update(ctx)
	if err == nil {
		p.lock.RLock()
		defer p.lock.RUnlock()
		for ch := range p.registry {
			ch <- &update
		}
		slog.Debug("poll completed", slog.Duration("duration", time.Since(start)))
	}
	return err
}

func (p *TadoPoller) update(ctx context.Context) (update Update, err error) {
	update.UserInfo, err = p.getMobileDevices(ctx)
	if err == nil {
		update.WeatherInfo, err = p.API.GetWeatherInfo(ctx)
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
	return
}

func (p *TadoPoller) getMobileDevices(ctx context.Context) (deviceMap map[int]tado.MobileDevice, err error) {
	var devices []tado.MobileDevice
	if devices, err = p.API.GetMobileDevices(ctx); err == nil {
		deviceMap = make(map[int]tado.MobileDevice)
		for _, device := range devices {
			if device.Settings.GeoTrackingEnabled {
				deviceMap[device.ID] = device
			}
		}
	}
	return
}

func (p *TadoPoller) getZones(ctx context.Context) (map[int]tado.Zone, error) {
	var zoneMap map[int]tado.Zone
	zones, err := p.API.GetZones(ctx)
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
		zoneInfo, err := p.API.GetZoneInfo(ctx, zoneID)
		if err != nil {
			return nil, err
		}

		zoneInfoMap[zoneID] = zoneInfo
	}
	return zoneInfoMap, nil
}

func (p *TadoPoller) getHomeState(ctx context.Context) (bool, error) {
	var home bool
	homeState, err := p.API.GetHomeState(ctx)
	if err == nil {
		home = homeState.Presence == "HOME"
	}
	return home, err
}
