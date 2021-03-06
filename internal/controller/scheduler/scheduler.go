package scheduler

import (
	"github.com/clambin/tado-exporter/pkg/tado"
)

// TadoData contains all data retrieved from Tado and needed to evaluate rules.
//
// This allows data to be shared between the scheduler and its clients without locking mechanisms: each client
// gets its own copy of the data without having to worry if its changed by a subsequent refresh
type TadoData struct {
	Zone         map[int]tado.Zone
	ZoneInfo     map[int]tado.ZoneInfo
	MobileDevice map[int]tado.MobileDevice
}

// UpdateChannel used by scheduler to send updates to the rules
type UpdateChannel chan *TadoData

// Scheduler is the heart of the controller.  On run, it updates all info from Tado
// and signals any registered rules to run
type Scheduler struct {
	clients []UpdateChannel
}

// Register a new client. This allocates a channel that the client should listen on for updates
func (scheduler *Scheduler) Register() (channel UpdateChannel) {
	if scheduler.clients == nil {
		scheduler.clients = make([]UpdateChannel, 0)
	}

	channel = make(UpdateChannel, 1)
	scheduler.clients = append(scheduler.clients, channel)
	return channel
}

// Notify all clients of the updated data
func (scheduler *Scheduler) Notify(tadoData TadoData) {
	for _, client := range scheduler.clients {
		client <- &tadoData
	}
}

// Stop signals all clients to shut down
func (scheduler *Scheduler) Stop() {
	for _, client := range scheduler.clients {
		client <- nil
	}
}
