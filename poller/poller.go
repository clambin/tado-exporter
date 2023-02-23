package poller

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	tadoAPI "github.com/clambin/tado-exporter/tado"
	"golang.org/x/exp/slog"
	"sync"
	"time"
)

//go:generate mockery --name Poller
type Poller interface {
	Run(ctx context.Context, interval time.Duration)
	Register() chan *Update
	Unregister(ch chan *Update)
	Refresh()
}

var _ Poller = &Server{}

type Server struct {
	API      tadoAPI.API
	refresh  chan struct{}
	registry map[chan *Update]struct{}
	lock     sync.RWMutex
}

func New(API tadoAPI.API) *Server {
	return &Server{
		API:      API,
		refresh:  make(chan struct{}),
		registry: make(map[chan *Update]struct{}),
	}
}

func (poller *Server) Run(ctx context.Context, interval time.Duration) {
	timer := time.NewTicker(interval)
	defer timer.Stop()

	slog.Info("poller started", "interval", interval)

	for {
		shouldPoll := false
		select {
		case <-ctx.Done():
			slog.Info("poller stopped")
			return
		case <-timer.C:
			shouldPoll = true
		case <-poller.refresh:
			shouldPoll = true
		}

		if shouldPoll {
			// poll for new data
			if err := poller.poll(ctx); err != nil {
				slog.Error("failed to get Tado metrics", err)
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
		slog.Debug("update received", "duration", time.Since(start))
		poller.lock.RLock()
		defer poller.lock.RUnlock()
		for ch := range poller.registry {
			ch <- &update
		}
		slog.Debug("update sent", "clients", len(poller.registry))
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

func (poller *Server) getZones(ctx context.Context) (zoneMap map[int]tado.Zone, err error) {
	var zones tado.Zones
	if zones, err = poller.API.GetZones(ctx); err == nil {
		zoneMap = make(map[int]tado.Zone)
		for _, zone := range zones {
			zoneMap[zone.ID] = zone
		}

	}
	return
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
	homeState, err := poller.API.GetHomeState(ctx)
	if err != nil {
		return false, err
	}
	return homeState.Presence == "HOME", nil
}
