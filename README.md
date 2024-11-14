# Tadoº exporter & controller
[![release](https://img.shields.io/github/v/tag/clambin/tado-exporter?color=green&label=release&style=plastic)](https://github.com/clambin/tado-exporter/releases)
[![codecov](https://img.shields.io/codecov/c/gh/clambin/tado-exporter?style=plastic)](https://app.codecov.io/gh/clambin/tado-exporter)
[![test](https://github.com/clambin/tado-exporter/workflows/test/badge.svg)](https://github.com/clambin/tado-exporter/actions)
[![build](https://github.com/clambin/tado-exporter/workflows/build/badge.svg)](https://github.com/clambin/tado-exporter/actions)
[![report card](https://goreportcard.com/badge/github.com/clambin/tado-exporter)](https://goreportcard.com/report/github.com/clambin/tado-exporter)
[![license](https://img.shields.io/github/license/clambin/tado-exporter?style=plastic)](LICENSE.md)

Monitor & control utility Tadoº Smart Thermostat devices.

## Features

tado retrieves all metrics from your Tadoº devices and makes them available to Prometheus. Additionally, tado can run:

- a Slack bot to query and control heating in a room
- a rule-based controller to set the heating based on current conditions, like:
  - switching on/off the heating in a room, when designated users are home or away
  - switching off a manual overlay after a specific amount of time
  - switching off a manual overlay at a specific time of the day
  - switching off all heating when all users are away (basic geofencing implementation)

## Installation

Container images for `tado monitor` are available on [ghcr.io](https://github.com/clambin/tado-exporter/pkgs/container/tado-monitor).

## Running
### Command-line options

The following command-line arguments are supported:

```
Usage:
tado [command]

Available Commands:
completion  Generate the autocompletion script for the specified shell
help        Help about any command
monitor     Monitor Tado thermostats
```

### Configuration file

TODO: app-token
The configuration file option specifies a yaml file to control the behaviour:

```
# Set to true to enable debug logging
debug: false
# Section for Prometheus exporter functionality
exporter:
    # Listener address for the Prometheus metrics server
    addr: :9090
# Section related to polling Tado for new metrics
poller:
    # How often we should poll for new metrics
    interval: 30s
# Section related to the /health endpoint
health:
    # Listener address for the /health endpoint
  addr: :8080
# Section containing Tado credentials
tado:
    username: ""
    password: ""
# Section for controller functionality
controller:
    tadobot:
        # When set, the controller will start a slack bot. See below for details
        enabled: true
        # Slackbot token value
        token: ""
```

If the filename isn't specified on the command line, the program looks for a file `config.yaml` in the following directories:

```
/etc/tado-monitor
$HOME/.tado-monitor
.
```

You can override any value in the configuration file by setting an environment variable with a prefix `TADO_MONITOR_`. 
For example, to avoid setting your Tadoº credentials in the configuration file, set the following environment variables:
s
```
export TADO_MONITOR_TADO.USERNAME="username@example.com"
export TADO_MONITOR_TADO.PASSWORD="your-password"
```

## Controlling your tadoº devices

`tado monitor` looks for a file `rules.yaml` in the same directory as the `config.yaml` file.
This file defines the rules to apply for your home:

```
# Home rules control the state of your home (i.e. "home" or "away").
home:
  # autoAway sets the home to "away" mode when all defined users are away from home.
  autoAway:
    delay: 1h
    users: [ "A", "B"]
# Zone rules control the state of a rooom within your home. Rules will either switch heating off when all users are away,
# or move the room to automatic mode when the room's been in manual mode for a while (think someone switching the bathroom
# to a manual temperature setting and then forgetting to switch it back to automatic mode).
zones:
  - name: "room 1"
    rules:
      # autoAway switches off the heating in a room when all defined users are away from home
      autoAway:
        delay: 1h
        users: ["A"]
      # limitOverlay removes a manual temperature control after a specified amount of time
      limitOverlay:
        delay: 1h
      # nightTime removes any manual temperature control at a specified time of day
      nightTime:
        time: "23:30:00"
```

If the file doesn't exist, `tado monitor` only runs as a Prometheus exporter.

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

### Grafana

The repo contains a sample [Grafana dashboard](assets/grafana/dashboards) that displays the scraped metrics. Feel free to customize as you see fit.

## Slack bot

`tado monitor` can run a Slack bot that reports on any rules being triggered:

![screenshot](assets/screenshots/tadobot_2.png?raw=true)

Users can also interact with the bot:

![screenshot](assets/screenshots/tadobot_1.png?raw=true)

The tado bot implements the following commands:

* **rules**: show any activated rules
* **rooms**: show temperature & settings on each room
* **users**: show the status of each user (home/away)
* **set**: sets the room's target temperature, optionally for a limited duration:
  * **set Bathroom 23.5**: sets the bathroom's target temperature to 23.5ºC
  * **set Bathroom 23.5 1h**: same, but switches back to automatic mode after 1 hour
  * **set Study auto**: sets the study to automatic temperature control
* **version**: show version
* **help**: show all available commands

To enable the bot, add a bot to the Custom Integrations of your Slack workspace and add the API Token in `controller.tadobot.token`.

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
