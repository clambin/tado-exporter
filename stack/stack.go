package stack

import (
	"context"
	"fmt"
	"github.com/clambin/go-metrics/server"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/collector"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
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

// Stack groups all components so they can be easily started/stopped
type Stack struct {
	Poller       poller.Poller
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
		Poller:       poller.New(API),
		MetricServer: server.New(cfg.Port),
		cfg:          cfg,
	}

	if stack.cfg.Exporter.Enabled {
		stack.Collector = collector.New()
	}

	if stack.cfg.Controller.Enabled {
		stack.TadoBot = slackbot.Create("tado "+version.BuildVersion, stack.cfg.Controller.TadoBot.Token, nil)
		stack.Controller = controller.New(API, &stack.cfg.Controller, stack.TadoBot, stack.Poller)
	}

	return
}

func (stack *Stack) Start(ctx context.Context) {
	stack.wg.Add(1)
	go func() {
		stack.Poller.Run(ctx, stack.cfg.Interval)
		stack.wg.Done()
	}()

	if stack.Collector != nil {
		stack.wg.Add(1)
		go func() {
			stack.Collector.Run(ctx)
			stack.wg.Done()
		}()

		stack.Poller.Register(stack.Collector.Update)
		prometheus.MustRegister(stack.Collector)

	}

	if stack.TadoBot != nil {
		stack.wg.Add(1)
		go func() {
			if err := stack.TadoBot.Run(ctx); err != nil {
				log.WithError(err).Fatal("tadoBot failed to start")
			}
			stack.wg.Done()
		}()
	}

	if stack.Controller != nil {
		stack.wg.Add(1)
		go func() {
			stack.Controller.Run(ctx, time.Minute)
			stack.wg.Done()
		}()
		stack.Poller.Register(stack.Controller.Updates)
	}

	stack.wg.Add(1)
	go func() {
		log.Info("HTTP server started")
		err2 := stack.MetricServer.Run()
		if err2 != http.ErrServerClosed {
			log.WithError(err2).Fatal("unable to start HTTP server")
		}
		log.Info("HTTP server stopped")
		stack.wg.Done()
	}()
}

func (stack *Stack) Stop() {
	err := stack.MetricServer.Shutdown(30 * time.Second)
	if err != nil {
		log.WithError(err).Warning("encountered error stopping HTTP Server")
	}
	stack.wg.Wait()
}
