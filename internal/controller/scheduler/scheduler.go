package scheduler

import (
	"github.com/clambin/tado-exporter/internal/tadobot"
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

// Scheduler is the heart of the controller.  On run, it updates all info from Tado
// and signals any registered rules to run
type Scheduler struct {
	TadoBot *tadobot.TadoBot
	clients []chan *TadoData
}

// Register a new client. This allocates a channel that the client should listen on for updates
func (scheduler *Scheduler) Register() (channel chan *TadoData) {
	if scheduler.clients == nil {
		scheduler.clients = make([]chan *TadoData, 0)
	}

	channel = make(chan *TadoData, 1)
	scheduler.clients = append(scheduler.clients, channel)
	return channel
}

// Run an update and signal any clients to evaluate their rules
func (scheduler *Scheduler) Run(tadoData *TadoData) (err error) {
	// Notify all clients
	for _, client := range scheduler.clients {
		clientTadoData := *tadoData
		client <- &clientTadoData
	}
	return
}

// Stop signals all clients to shut down
func (scheduler *Scheduler) Stop() {
	for _, client := range scheduler.clients {
		client <- nil
	}
}

func (scheduler *Scheduler) Notify(title, message string) (err error) {
	if scheduler.TadoBot != nil {
		err = scheduler.TadoBot.SendMessage(title, message)
	}
	return
}
