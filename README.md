# Tadoº exporter
[![release](https://img.shields.io/github/v/tag/clambin/tado-exporter?color=green&label=release&style=plastic)](https://github.com/clambin/tado-exporter/releases)
[![codecov](https://img.shields.io/codecov/c/gh/clambin/tado-exporter?style=plastic)](https://app.codecov.io/gh/clambin/tado-exporter)
[![build](https://github.com/clambin/tado-exporter/workflows/build/badge.svg)](https://github.com/clambin/tado-exporter/actions)
[![report card](https://goreportcard.com/badge/github.com/clambin/tado-exporter)](https://goreportcard.com/report/github.com/clambin/tado-exporter)
[![license](https://img.shields.io/github/license/clambin/tado-exporter?style=plastic)](LICENSE.md)

tado-export exports Tadoº Smart Thermostat metrics to Prometheus.

## :warning: Breaking change in v0.19.0
As of v0.19.0, tado-exporter only supports exporting metrics to Prometheus. The more advances functions
(controlling the heating, the slack interface, etc.) have been moved to a separate [proteus](https://codeberg.org/clambin/proteus) project.

## :warning: Tadoº is lowering its rate limits
Tadoº has recently announced they will be lowering the rate limits for their API. 
As a result, tado-exporter may start to fail with a 429 error.

tado-exporter makes no effort to handle this or reduce the number of requests it makes.
It is recommended to reduce the scrape interval in your Prometheus configuration.

## Installation

Container images are available on [ghcr.io](https://github.com/clambin/tado-exporter/pkgs/container/tado-exporter).

## Running
### Command-line options

The following command-line arguments are supported:

```
Usage:
  -log.format string
        log format (default "text")
  -log.level string
        log level (default "info")
  -prom.addr string
        prometheus listen address (default ":9100")
  -prom.path string
        prometheus path (default "/metrics")
  -token.passphrase string
        passphrase to encrypt the token
  -token.path string
        path to store the (encrypted) token```
```

## Prometheus

### Adding tado as a target

Add tado as a target to let Prometheus scrape the metrics into its database.
This highly depends on your particular Prometheus configuration. In its simplest form, add a new scrape target to `prometheus.yml`:

```
scrape_configs:
- job_name: tado
  static_configs:
  - targets: [ '<tado host>:<port>' ]
```

where `port` is the Prometheus listener port configured in `exporter.addr`.

### Metrics

See [METRICS.md](METRICS.md) for details.

## Tadoº client implementation

tado uses the Tadoº Go Client found at [GitHub](https://github.com/clambin/tado). Feel free to reuse for your own projects.

## Authors

* **Christophe Lambin**

## Acknowledgements

* [tado OpenAPI specification](https://github.com/kritsel/tado-openapispec-v2) by [Kristel](https://github.com/kritsel).
* Max Rosin for his [Python implementation](https://github.com/ekeih/libtado) of the Tado API
* [vide/tado-exporter](https://github.com/vide/tado-exporter) for some inspiration

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.
