package poller

import (
	"context"
	"github.com/clambin/tado"
	log "github.com/sirupsen/logrus"
	"time"
)

//go:generate mockery --name Poller
type Poller interface {
	Refresh()
}

type Server struct {
	tado.API
	Register chan chan *Update
	refresh  chan struct{}
	registry []chan *Update
}

func New(API tado.API) *Server {
	return &Server{
		API:      API,
		Register: make(chan chan *Update),
		refresh:  make(chan struct{}),
		registry: make([]chan *Update, 0),
	}
}

func (poller *Server) Run(ctx context.Context, interval time.Duration) {
	var err error
	timer := time.NewTicker(10 * time.Millisecond)
	first := true

	log.Info("poller started")

	for running := true; running; {
		poll := false
		select {
		case <-ctx.Done():
			running = false
		case ch := <-poller.Register:
			poller.registry = append(poller.registry, ch)
		case <-timer.C:
			poll = true
		case <-poller.refresh:
			poll = true
		}

		// is anybody listening?
		if running && poll && len(poller.registry) > 0 {
			// poll for new data
			err = poller.Poll(ctx)
			if err != nil {
				log.WithError(err).Warning("failed to get Tado metrics")
			}
			// once we have registered listeners, poll at the desired interval
			if first {
				timer.Stop()
				timer = time.NewTicker(interval)
				first = false
			}
		}
	}
	timer.Stop()

	log.Info("poller stopped")
}

func (poller *Server) Refresh() {
	poller.refresh <- struct{}{}
}

func (poller *Server) Poll(ctx context.Context) (err error) {
	var update Update
	update, err = poller.update(ctx)

	if err == nil {
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
			if device.Settings.GeoTrackingEnabled == true {
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
