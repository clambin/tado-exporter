package poller

import (
	"context"
	"fmt"
	"github.com/clambin/go-common/taskmanager"
	"github.com/clambin/tado"
	"golang.org/x/exp/slog"
	"sync"
	"time"
)

//go:generate mockery --name Poller
type Poller interface {
	taskmanager.Task
	//Run(ctx context.Context) error
	Register() chan *Update
	Unregister(ch chan *Update)
	Refresh()
}

//go:generate mockery --name TadoGetter
type TadoGetter interface {
	GetWeatherInfo(context.Context) (tado.WeatherInfo, error)
	GetMobileDevices(context.Context) ([]tado.MobileDevice, error)
	GetZones(context.Context) (tado.Zones, error)
	GetZoneInfo(context.Context, int) (tado.ZoneInfo, error)
	GetHomeState(ctx context.Context) (homeState tado.HomeState, err error)
}

var _ Poller = &Server{}

type Server struct {
	API      TadoGetter
	interval time.Duration
	refresh  chan struct{}
	registry map[chan *Update]struct{}
	lock     sync.RWMutex
}

func New(API TadoGetter, interval time.Duration) *Server {
	return &Server{
		API:      API,
		interval: interval,
		refresh:  make(chan struct{}),
		registry: make(map[chan *Update]struct{}),
	}
}

func (p *Server) Run(ctx context.Context) error {
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

func (poller *Server) Refresh() {
	poller.refresh <- struct{}{}
}

func (poller *Server) Register() chan *Update {
	poller.lock.Lock()
	defer poller.lock.Unlock()
	ch := make(chan *Update, 1)
	poller.registry[ch] = struct{}{}
	slog.Debug(fmt.Sprintf("poller has %d clients", len(poller.registry)))
	return ch
}

func (poller *Server) Unregister(ch chan *Update) {
	poller.lock.Lock()
	defer poller.lock.Unlock()
	delete(poller.registry, ch)
	slog.Debug(fmt.Sprintf("poller has %d clients", len(poller.registry)))
}

func (poller *Server) poll(ctx context.Context) error {
	start := time.Now()
	update, err := poller.update(ctx)
	if err == nil {
		poller.lock.RLock()
		defer poller.lock.RUnlock()
		for ch := range poller.registry {
			ch <- &update
		}
		slog.Debug("poll completed", slog.Duration("duration", time.Since(start)))
	}
	return err
}

func (poller *Server) update(ctx context.Context) (update Update, err error) {
	update.UserInfo, err = poller.getMobileDevices(ctx)
	if err == nil {
		update.WeatherInfo, err = poller.API.GetWeatherInfo(ctx)
	}
	if err == nil {
		update.Zones, err = poller.getZones(ctx)
	}
	if err == nil {
		update.ZoneInfo, err = poller.getZoneInfos(ctx, update.Zones)
	}
	if err == nil {
		update.Home, err = poller.getHomeState(ctx)
	}
	return
}

func (poller *Server) getMobileDevices(ctx context.Context) (deviceMap map[int]tado.MobileDevice, err error) {
	var devices []tado.MobileDevice
	if devices, err = poller.API.GetMobileDevices(ctx); err == nil {
		deviceMap = make(map[int]tado.MobileDevice)
		for _, device := range devices {
			if device.Settings.GeoTrackingEnabled {
				deviceMap[device.ID] = device
			}
		}
	}
	return
}

func (poller *Server) getZones(ctx context.Context) (map[int]tado.Zone, error) {
	var zoneMap map[int]tado.Zone
	zones, err := poller.API.GetZones(ctx)
	if err == nil {
		zoneMap = make(map[int]tado.Zone)
		for _, zone := range zones {
			zoneMap[zone.ID] = zone
		}
	}
	return zoneMap, err
}

func (poller *Server) getZoneInfos(ctx context.Context, zones map[int]tado.Zone) (map[int]tado.ZoneInfo, error) {
	zoneInfoMap := make(map[int]tado.ZoneInfo)
	for zoneID := range zones {
		zoneInfo, err := poller.API.GetZoneInfo(ctx, zoneID)
		if err != nil {
			return nil, err
		}

		zoneInfoMap[zoneID] = zoneInfo
	}
	return zoneInfoMap, nil
}

func (poller *Server) getHomeState(ctx context.Context) (bool, error) {
	var home bool
	homeState, err := poller.API.GetHomeState(ctx)
	if err == nil {
		home = homeState.Presence == "HOME"
	}
	return home, err
}
