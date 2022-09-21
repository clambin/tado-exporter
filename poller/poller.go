package poller

import (
	"context"
	"github.com/clambin/tado"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

//go:generate mockery --name Poller
type Poller interface {
	Run(ctx context.Context, interval time.Duration)
	Register(ch chan *Update)
	Unregister(ch chan *Update)
	Refresh()
	GetLastUpdate() time.Time
}

var _ Poller = &Server{}

type Server struct {
	tado.API
	refresh    chan struct{}
	lastUpdate time.Time
	registry   map[chan *Update]struct{}
	lock       sync.RWMutex
}

func New(API tado.API) *Server {
	return &Server{
		API:      API,
		refresh:  make(chan struct{}),
		registry: make(map[chan *Update]struct{}),
	}
}

func (poller *Server) Run(ctx context.Context, interval time.Duration) {
	timer := time.NewTicker(interval)

	log.WithField("interval", interval).Info("poller started")

	for running := true; running; {
		poll := false
		select {
		case <-ctx.Done():
			running = false
		case <-timer.C:
			poll = true
		case <-poller.refresh:
			poll = true
		}

		if !poll {
			continue
		}

		// poll for new data
		if err := poller.poll(ctx); err != nil {
			log.WithError(err).Warning("failed to get Tado metrics")
		}
		poller.lock.Lock()
		poller.lastUpdate = time.Now()
		poller.lock.Unlock()
	}
	timer.Stop()

	log.Info("poller stopped")
}

func (poller *Server) Refresh() {
	poller.refresh <- struct{}{}
}

func (poller *Server) Register(ch chan *Update) {
	poller.lock.Lock()
	defer poller.lock.Unlock()
	poller.registry[ch] = struct{}{}
	log.Debugf("poller has %d clients", len(poller.registry))
}

func (poller *Server) Unregister(ch chan *Update) {
	poller.lock.Lock()
	defer poller.lock.Unlock()
	delete(poller.registry, ch)
	log.Debugf("poller has %d clients", len(poller.registry))
}

func (poller *Server) poll(ctx context.Context) error {
	update, err := poller.update(ctx)
	if err == nil {
		poller.lock.RLock()
		defer poller.lock.RUnlock()
		log.Debugf("sending update to %d registered clients", len(poller.registry))
		for ch := range poller.registry {
			ch <- &update
		}
		log.Debugf("sent update to %d registered clients", len(poller.registry))
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
	var zones []tado.Zone
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

func (poller *Server) GetLastUpdate() time.Time {
	poller.lock.RLock()
	defer poller.lock.RUnlock()
	return poller.lastUpdate
}
