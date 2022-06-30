package poller

import (
	"context"
	"github.com/clambin/tado"
	log "github.com/sirupsen/logrus"
	"time"
)

//go:generate mockery --name Poller
type Poller interface {
	Run(ctx context.Context, interval time.Duration)
	Refresh()
	Register(ch chan *Update)
}

var _ Poller = &Server{}

type Server struct {
	tado.API
	register chan chan *Update
	refresh  chan struct{}
	registry []chan *Update
}

func New(API tado.API) *Server {
	return &Server{
		API:      API,
		register: make(chan chan *Update),
		refresh:  make(chan struct{}),
		registry: make([]chan *Update, 0),
	}
}

func (poller *Server) Run(ctx context.Context, interval time.Duration) {
	var err error
	timer := time.NewTicker(interval)

	log.WithField("interval", interval).Info("poller started")

	for running := true; running; {
		poll := false
		select {
		case <-ctx.Done():
			running = false
		case ch := <-poller.register:
			poller.registry = append(poller.registry, ch)
			log.Debugf("poller registered new client. total clients: %d", len(poller.registry))
			// registration typically happens when we start up. so, let's poll for data so the client doesn't
			// need to wait for interval to expire before getting data
			poll = true
		case <-timer.C:
			poll = true
		case <-poller.refresh:
			poll = true
		}

		if running && poll {
			// poll for new data
			if err = poller.Poll(ctx); err != nil {
				log.WithError(err).Warning("failed to get Tado metrics")
			}

		}
	}
	timer.Stop()

	log.Info("poller stopped")
}

func (poller *Server) Refresh() {
	poller.refresh <- struct{}{}
}

func (poller *Server) Register(ch chan *Update) {
	poller.register <- ch
}

func (poller *Server) Poll(ctx context.Context) (err error) {
	// is anybody listening?
	if len(poller.registry) == 0 {
		log.Debug("poller has no clients. skipping update")
		return
	}

	var update Update
	if update, err = poller.update(ctx); err == nil {
		for _, ch := range poller.registry {
			ch <- &update
		}
	}
	return
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
	deviceMap = make(map[int]tado.MobileDevice)

	var devices []tado.MobileDevice
	devices, err = poller.API.GetMobileDevices(ctx)

	if err == nil {
		for _, device := range devices {
			if device.Settings.GeoTrackingEnabled {
				deviceMap[device.ID] = device
			}
		}
	}

	return
}

func (poller *Server) getZones(ctx context.Context) (zoneMap map[int]tado.Zone, err error) {
	zoneMap = make(map[int]tado.Zone)

	var zones []tado.Zone
	zones, err = poller.API.GetZones(ctx)

	if err == nil {
		for _, zone := range zones {
			zoneMap[zone.ID] = zone
		}
	}
	return
}

func (poller *Server) getZoneInfos(ctx context.Context, zones map[int]tado.Zone) (zoneInfoMap map[int]tado.ZoneInfo, err error) {
	zoneInfoMap = make(map[int]tado.ZoneInfo)

	for zoneID := range zones {
		var zoneInfo tado.ZoneInfo
		zoneInfo, err = poller.API.GetZoneInfo(ctx, zoneID)

		if err == nil {
			zoneInfoMap[zoneID] = zoneInfo
		}
	}

	return
}
