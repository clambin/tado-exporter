# Prometheus Metrics

tado exposes the following metrics:

## General metrics

| metric | type |  labels | help |
| --- | --- |  --- | --- |
| tado_home_state | GAUGE | home_state|State of the home. Always 1. Label home_state specifies the state |
| tado_outside_temp_celsius | GAUGE | |Current outside temperature in degrees celsius |
| tado_solar_intensity_percentage | GAUGE | |Current solar intensity in percentage (0-100) |
| tado_weather | GAUGE | tado_weather|Current weather. Always one. See label 'tado_weather' |

## Metrics by Zone

The following metrics are reported for each discovered zone. The zone name is added as 'zone_name' label.

| metric | type |  labels | help |
| --- | --- |  --- | --- |
| tado_zone_device_battery_status | GAUGE | id, type, zone_name|Tado device battery status |
| tado_zone_device_connection_status | GAUGE | firmware, id, type, zone_name|Tado device connection status |
| tado_zone_heating_percentage | GAUGE | zone_name|Current heating percentage in this zone in percentage (0-100) |
| tado_zone_humidity_percentage | GAUGE | zone_name|Current humidity percentage in this zone in percentage (0-100) |
| tado_zone_open_window_duration | GAUGE | zone_name|Duration of open window event in seconds |
| tado_zone_open_window_remaining | GAUGE | zone_name|Remaining duration of open window event in seconds |
| tado_zone_power_state | GAUGE | zone_name|Power status of this zone |
| tado_zone_target_manual_mode | GAUGE | zone_name|1 if this zone is in manual temp target mode |
| tado_zone_target_temp_celsius | GAUGE | zone_name|Target temperature of this zone in degrees celsius |
| tado_zone_temperature_celsius | GAUGE | zone_name|Current temperature of this zone in degrees celsius |

## Mobile device home/away status metrics

Tado reports the home/away status of registered mobile devices. See device name is added as 'name' label.

| metric | type |  labels | help |
| --- | --- |  --- | --- |
| tado_mobile_device_status | GAUGE | name|Tado mobile device status. 1 if the device is "home" |
