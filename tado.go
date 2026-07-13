package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"codeberg.org/clambin/go-common/flagger"
	"codeberg.org/clambin/proteus/integrations/climate"
	"codeberg.org/clambin/proteus/integrations/climate/tado"
	tado2 "github.com/clambin/tado/v2"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/oauth2"
)

var (
	// overridden during build
	version = "change-me"
)

type configuration struct {
	flagger.Log
	flagger.Prom
	Token
}

type Token struct {
	Path       string `flagger.usage:"path to store the (encrypted) token"`
	Passphrase string `flagger.usage:"passphrase to encrypt the token"`
}

func main() {
	cfg := configuration{
		Log:  flagger.DefaultLog,
		Prom: flagger.DefaultProm,
	}
	flagger.SetFlags(flag.CommandLine, &cfg)
	flag.Parse()
	logger := cfg.Logger(os.Stderr, nil)
	if cfg.Token.Path == "" {
		logger.Error("token storage path is required")
		os.Exit(1)
	}
	if cfg.Passphrase == "" {
		logger.Error("token storage passphrase is required")
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client, err := makeTadoClient(ctx, cfg.Token.Path, cfg.Passphrase)
	if err != nil {
		logger.Error("failed to create Tado client", "err", err)
		os.Exit(1)
	}

	logger.Debug("created Tado client", "tokenPath", cfg.Token.Path)

	c := collector{
		scraper: &tado.Scraper{
			Client:      client,
			Logger:      logger,
			Descriptors: climate.NewMetricDescriptors("tado"),
		},
	}

	prometheus.MustRegister(c)

	logger.Info("tado exporter starting", "version", version)

	if err := cfg.Serve(ctx); err != nil {
		logger.Error("error starting prometheus agent", "err", err)
	}
}

func makeTadoClient(
	ctx context.Context,
	path string,
	passphrase string,
) (*tado2.ClientWithResponses, error) {
	// create an oauth2 http client that uses the device auth flow
	httpClient, err := tado2.NewOAuth2Client(ctx, path, passphrase, func(response *oauth2.DeviceAuthResponse) {
		fmt.Printf("No token found. Visit %s and log in ...\n", response.VerificationURIComplete)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create oauth2 client: %w", err)
	}
	// create the tado client using the oauth2 http client
	return tado2.NewClientWithResponses(tado2.ServerURL, tado2.WithHTTPClient(httpClient))
}

var _ prometheus.Collector = collector{}

type collector struct {
	scraper *tado.Scraper
}

func (c collector) Describe(ch chan<- *prometheus.Desc) {
	c.scraper.Descriptors.Describe(ch)
}

func (c collector) Collect(ch chan<- prometheus.Metric) {
	metrics, err := c.scraper.Scrape(context.Background())
	if err != nil {
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("tado_scrape_error", "error scraping tado", nil, nil), err)
		return
	}
	metrics.Collect(ch)
}
