package stack

import (
	"context"
	"fmt"
	"github.com/clambin/go-metrics/server"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/collector"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/health"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/version"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"sync"
	"time"
)

// Stack groups all components, so they can be easily started/stopped
type Stack struct {
	Poller       poller.Poller
	Health       *health.Handler
	Collector    *collector.Collector
	TadoBot      slackbot.SlackBot
	Controller   *controller.Controller
	MetricServer *server.Server
	cfg          *configuration.Configuration
	wg           sync.WaitGroup
}

func New(cfg *configuration.Configuration) (stack *Stack, err error) {
	username := os.Getenv("TADO_USERNAME")
	password := os.Getenv("TADO_PASSWORD")
	clientSecret := os.Getenv("TADO_CLIENT_SECRET")

	if username == "" || password == "" {
		return nil, fmt.Errorf("TADO_USERNAME/TADO_PASSWORD environment variables not set")
	}

	API := tado.New(username, password, clientSecret)
	stack = &Stack{
		Poller: poller.New(API),
		cfg:    cfg,
	}

	stack.Health = &health.Handler{Poller: stack.Poller, Ch: make(chan *poller.Update)}

	stack.MetricServer = server.NewWithHandlers(cfg.Port, []server.Handler{
		{Path: "/health", Handler: http.HandlerFunc(stack.Health.Handle)},
	})

	if stack.cfg.Exporter.Enabled {
		stack.Collector = collector.New(stack.Poller)
	}

	if stack.cfg.Controller.Enabled {
		stack.TadoBot = slackbot.Create("tado "+version.BuildVersion, stack.cfg.Controller.TadoBot.Token, nil)
		stack.Controller = controller.New(API, &stack.cfg.Controller, stack.TadoBot, stack.Poller)
	}

	return
}

func (s *Stack) Start(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		s.Poller.Run(ctx, s.cfg.Interval)
		s.wg.Done()
	}()

	s.wg.Add(1)
	go func() {
		s.Health.Run(ctx)
		s.wg.Done()
	}()

	if s.Collector != nil {
		s.wg.Add(1)
		go func() {
			s.Collector.Run(ctx)
			s.wg.Done()
		}()
		prometheus.MustRegister(s.Collector)
	}

	if s.TadoBot != nil {
		s.wg.Add(1)
		go func() {
			if err := s.TadoBot.Run(ctx); err != nil {
				log.WithError(err).Fatal("tadoBot failed to start")
			}
			s.wg.Done()
		}()
	}

	if s.Controller != nil {
		s.wg.Add(1)
		go func() {
			s.Controller.Run(ctx, time.Minute)
			s.wg.Done()
		}()
	}

	s.wg.Add(1)
	go func() {
		log.Info("HTTP server started")
		err2 := s.MetricServer.Run()
		if err2 != http.ErrServerClosed {
			log.WithError(err2).Fatal("unable to start HTTP server")
		}
		log.Info("HTTP server stopped")
		s.wg.Done()
	}()

	//s.Poller.Refresh()
}

func (s *Stack) Stop() {
	err := s.MetricServer.Shutdown(30 * time.Second)
	if err != nil {
		log.WithError(err).Warning("encountered error stopping HTTP Server")
	}
	s.wg.Wait()
}
