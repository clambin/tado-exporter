# tado-exporter
![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/clambin/tado-exporter?color=green&label=Release&style=plastic)
![Codecov](https://img.shields.io/codecov/c/gh/clambin/tado-exporter?style=plastic)
![Build](https://github.com/clambin/tado-exporter/workflows/Build/badge.svg)
![Go Report Card](https://goreportcard.com/badge/github.com/clambin/tado-exporter)
![GitHub](https://img.shields.io/github/license/clambin/tado-exporter?style=plastic)

Prometheus exporter for Tadoº Smart Thermostat devices.

## Installation

A Docker image is available on [docker](https://hub.docker.com/clambin/tado-exporter).  Images are available for amd64 & arm32v7.

Alternatively, you can clone the repository from [github](https://github.com/r/clambin/tado-exporter) and build from source:

```
git clone https://github.com/clambin/tado-exporter.git
cd tado-exporter
go build
```

You will need to have go 1.15 installed on your system.

## Running tado-exporter

Set the following environment variables prior to starting tado-exporter:

```
* TADO_USERNAME: your Tado username
* TADO_PASSWORD: your Tado password
```

In case you run into authentication problems, you may need to retrieve your `CLIENT_SECRET` and export this environment variable as well.
To get your `CLIENT_SECRET`, log into tado.com and visit [https://my.tado.com/webapp/env.js](https://my.tado.com/webapp/env.js).
The client secret can be found in the oauth section:

```
var TD = {
	config: {
...
		oauth: {
...
			clientSecret: 'verylongclientsecret'
		}
	}
};
```

Once you have the relevant environment variables set, you can start tado-exporter. E.g. when using Docker, start tado-exporter as follows:

```
docker run -e TADO_USERNAME=user@example.com -e TADO_PASSWORD="your-password" --rm -p 8080:8080 clambin/tado-exporter:latest
```

### Prometheus

Add tado-exporter as a target to let Prometheus scrape the metrics into its database. 
This highly depends on your particular Prometheus configuration. In it simplest form, add a new scrape target to `prometheus.yml`:

```
scrape_configs:
- job_name: tado
  static_configs:
  - targets: [ '<tado host>:8080' ]
```

### Command-line options

The following command-line arguments can be passed:

```
usage: tado-exporter [<flags>]

tado-exporter

Flags:
  -h, --help         Show context-sensitive help (also try --help-long and --help-man).
  -v, --version      Show application version.
      --debug        Log debug messages
      --port=8080    API listener port
      --interval=1m  Scrape interval

```

### Metrics

tado-exporter exposes the following metrics:

```
* tado_zone_target_temp_celsius:   Target temperature of this zone in degrees celsius
* tado_zone_power_state:           Power status of this zone
* tado_device_connection_status:   Connection status of devices in this zone
* tado_device_battery_status:      Battery status of devices in this zone
* tado_temperature_celsius:        Current temperature of this zone in degrees celsius
* tado_heating_percentage:         Current heating percentage in this zone in percentage (0-100)
* tado_humidity_percentage:        Current humidity percentage in this zone
* tado_outside_temp_celsius:       Current outside temperature in degrees celsius
* tado_solar_intensity_percentage: Current solar intensity in percentage (0-100)
* tado_weather:                    Current weather. Always one. See label 'tado_weather'
```

### Grafana

[Github](https://github.com/clambin/tado-exporter/assets/grafana/dashboards) contains a sample Grafana dashboard to visualize the scraped metrics.
Feel free to customize as you see fit.

## Authors

* **Christophe Lambin**

## Acknowledgements

* Max Rosin for his [Python implementation](https://github.com/ekeih/libtado) of the Tado API
* [vide/tado-exporter](https://github.com/vide/tado-exporter) for some inspiration


## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.
