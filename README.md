# tado-monitor
![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/clambin/tado-exporter?color=green&label=Release&style=plastic)
![Codecov](https://img.shields.io/codecov/c/gh/clambin/tado-exporter?style=plastic)
![Build](https://github.com/clambin/tado-exporter/workflows/Build/badge.svg)
![Go Report Card](https://goreportcard.com/badge/github.com/clambin/tado-exporter)
![GitHub](https://img.shields.io/github/license/clambin/tado-exporter?style=plastic)

Monitor & control utility Tadoº Smart Thermostat devices.

## Features

tado-monitor offers two types of functionality:

* an exporter to expose metrics to Prometheus
* a controller to control the temperature settings of rooms in your Tado-controlled home

The controller is rule-based. It currently supports three types of rules:

* **autoAway** rules switch off the automatic temperature control for a room when a user has left home for a specified amount of time. Typical use case is one of the kids leaving for college for the week.
* **limitOverlay** disables manual temperature settings in a room after a specified amount of time. Typical use case is to disable the bathroom heating if someone forgot to switch off a manual temperature setting.
* **nightTime** disables manual temperature settings in a room at a specific time of day. Typical use case is switching the living room to automatic at the end of the day.


## Installation

Binaries are available on the [release](https://github.com/clambin/tado-exporter/releases) page. Docker images are available on [ghcr.io](https://github.com/clambin/tado-exporter/pkgs/container/tado-monitor).

Alternatively, you can clone the repository and build from source:

```
git clone https://github.com/clambin/tado-exporter.git
cd tado-exporter
go build tado-monitor
```

You will need to have Go 1.15 installed on your system.

## Running tado-monitor
### Command-line options

The following command-line arguments can be passed:

```
usage: tado-monitor --config=CONFIG [<flags>]

tado-monitor

Flags:
  -h, --help           Show context-sensitive help (also try --help-long and --help-man).
  -v, --version        Show application version.
      --debug          Log debug messages
      --config=CONFIG  Configuration file
```

### Tadoº credentials
Set the following environment variables prior to starting tado-monitor:

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

### Configuration file
The (mandatory) configuration file option specifies a yaml file to control tado-monitor's behaviour:

```
# Set to true to enable debug logging
debug: false

# Section for Prometheus exporter functionality
exporter:
  # Enable exporter functionality
  enabled: true
  # HTTP port for Prometheus scraping. Metrics are exposed on /metrics path
  port: 8080
  # How often the exporter should run
  interval: 1m
  
# Section for controller functionality
controller:
  # Enable controller functionality
  enabled: false
  # How often rules should be evaluated
  interval: 5m
  tadoBot:
    # When set, tado will start a slack bot.  See below for details.
    enabled: false
    token:
      # slackbot token value
      value: ""
      # if set, tado will use the specified environment variable's value as slackbot token.
      # Overrides the value above.
      envVar: ""
    # Zones defined rules for individual rooms (i.e. "zones")
    zones:
    - name: "zone name"
      # autoAway switches off heating in a room when all associated users are not home
      autoAway:
        enabled: false
        delay: 1h
        # users specifies the list of users assocated with that room. 
        # either name or id can be used
        users:
          - name: "my Phone"
      # limitOverlay switches off a room's manual temperature setting after a configured amount of time
      limitOverlay:
        enabled: false
        delay: 10m
      # nighttime sets a room to automatic temperature control at a fixed time of day
      nightTime:
        enabled: false
        time: "23:30:00"

```

## Prometheus

### Adding tado-monitor as a target

Add tado-exporter as a target to let Prometheus scrape the metrics into its database.
This highly depends on your particular Prometheus configuration. In its simplest form, add a new scrape target to `prometheus.yml`:

```
scrape_configs:
- job_name: tado
  static_configs:
  - targets: [ '<tado host>:8080' ]
```
### Metrics

tado-exporter exposes the following metrics:

#### Metrics by Zone

The following metrics are reported for each discovered zone.  The zone name is added as 'zone_name' label.

```
* tado_zone_device_battery_status:      Battery status of devices in this zone
* tado_zone_device_connection_status:   Connection status of devices in this zone
* tado_zone_heating_percentage:         Current heating percentage in this zone in percentage (0-100)
* tado_zone_humidity_percentage:        Current humidity percentage in this zone
* tado_zone_open_window_duration:       Duration of open window event in seconds
* tado_zone_open_window_remaining:      Remaining duration of open window event in seconds
* tado_zone_power_state:                Power status of this zone
* tado_zone_target_manual_mode:         1.0 if this zone is in manual target temp mode
* tado_zone_target_temp_celsius:        Target temperature of this zone in degrees celsius
* tado_zone_temperature_celsius:        Current temperature of this zone in degrees celsius
```

#### Mobile device home/away status metrics

Tado reports the home/away status of registered mobile devices. See device name is added as 'name' label.

```
* tado_mobile_device_status:            Status of any geotracked mobile devices (1: at home, 0: away)

```

#### General metrics

```
* tado_outside_temp_celsius:            Current outside temperature in degrees celsius
* tado_solar_intensity_percentage:      Current solar intensity in percentage (0-100)
* tado_weather:                         Current weather. Always one. See label 'tado_weather'
```

### Grafana

The repo contains a sample [Grafana dashboard](assets/grafana/dashboards) to visualize the scraped metrics.

Feel free to customize as you see fit.

## Slack bot

Tado-controller can run a slack bot that will report on any rules being triggered:

![screenshot](assets/screenshots/tadobot_2.png?raw=true)

Users can also interact with the bot:

![screenshot](assets/screenshots/tadobot_1.png?raw=true)

TadoBot supports the following commands:

* **rules**: show any activated rules
* **rooms**: show temperature & settings on each room
* **set**: sets the room's target temperature, optionally for a limited duration:
  * **set Bathroom 23.5**: sets the bathroom's target temperature to 23.5ºC
  * **set Bathroom 23.5 1h**: same, but switches back to automatic mode after 1 hour
  * **set Study auto**: sets the study to automatic temperature control
* **version**: show version of tado-monitor
* **help**: show all available commands

To enable the bot, add a bot to the workspace's Custom Integrations and add the API Token in the configuration file above (*slackbotToken*).

## Tado Client API

The Tado Client API implementation can be found in [github](https://github.com/clambin/tado). The API should be fairly stable at this point, 
so feel free to reuse for your own projects.

## Authors

* **Christophe Lambin**

## Acknowledgements

* Max Rosin for his [Python implementation](https://github.com/ekeih/libtado) of the Tado API
* [vide/tado-exporter](https://github.com/vide/tado-exporter) for some inspiration


## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.
