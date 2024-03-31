package monitor

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/cmd/monitor/mocks"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"
)

func Test_maybeLoadRules(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		wantErr assert.ErrorAssertionFunc
		want    configuration.Configuration
	}{
		{
			name: "valid",
			content: `zones:
  - name: "bathroom"
    rules:
      limitOverlay:
        delay: 1h
`,
			wantErr: assert.NoError,
			want: configuration.Configuration{
				Zones: []configuration.ZoneConfiguration{
					{
						Name: "bathroom",
						Rules: configuration.ZoneRuleConfiguration{
							LimitOverlay: configuration.LimitOverlayConfiguration{Delay: time.Hour},
						},
					},
				},
			},
		},
		{
			name:    "invalid",
			content: `invalid yaml`,
			wantErr: assert.Error,
		},
		{
			name:    "missing",
			content: ``,
			wantErr: assert.NoError,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f, err := os.CreateTemp("", "")
			require.NoError(t, err)

			if tt.content != "" {
				_, err := f.Write([]byte(tt.content))
				require.NoError(t, err)
				_ = f.Close()
				defer func() { _ = os.Remove(f.Name()) }()
			} else {
				_ = f.Close()
				_ = os.Remove(f.Name())
			}

			r, err := maybeLoadRules(f.Name())
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, r)
		})
	}
}

func Test_runMonitor(t *testing.T) {
	r := prometheus.NewPedanticRegistry()
	v := viper.New()
	v.Set("poller.interval", "1m")
	v.Set("exporter.addr", ":9090")
	v.Set("health.addr", ":8080")
	l := slog.Default()

	ctx, cancel := context.WithCancel(context.Background())

	client := mocks.NewTadoClient(t)
	client.EXPECT().GetMobileDevices(ctx).Return([]tado.MobileDevice{{
		ID:       1,
		Name:     "foo",
		Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
		Location: tado.MobileDeviceLocation{AtHome: true},
	}}, nil)
	client.EXPECT().GetWeatherInfo(ctx).Return(tado.WeatherInfo{
		OutsideTemperature: tado.Temperature{Celsius: 25},
		SolarIntensity:     tado.Percentage{Percentage: 80},
		WeatherState:       tado.Value{Value: "SUNNY"},
	}, nil)
	client.EXPECT().GetZones(ctx).Return(tado.Zones{{ID: 1, Name: "foo"}}, nil)
	client.EXPECT().GetZoneInfo(ctx, 1).Return(tado.ZoneInfo{
		Setting:            tado.ZonePowerSetting{Temperature: tado.Temperature{Celsius: 22}},
		ActivityDataPoints: tado.ZoneInfoActivityDataPoints{HeatingPower: tado.Percentage{Percentage: 25.0}},
		SensorDataPoints: tado.ZoneInfoSensorDataPoints{
			InsideTemperature: tado.Temperature{Celsius: 21},
			Humidity:          tado.Percentage{Percentage: 55},
		},
	}, nil)
	client.EXPECT().GetHomeState(ctx).Return(tado.HomeState{Presence: "HOME"}, nil)

	errCh := make(chan error)
	go func() { errCh <- runMonitor(ctx, l, v, r, client, "1.0") }()

	assert.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:9090/metrics")
		return err == nil && resp.StatusCode == http.StatusOK
	}, time.Second, 50*time.Millisecond)

	assert.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:8080/health")
		return err == nil && resp.StatusCode == http.StatusOK
	}, time.Second, 50*time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}
