package monitor

import (
	"context"
	"github.com/clambin/tado"
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
	errCh := make(chan error)
	go func() { errCh <- runMonitor(ctx, l, v, r, fakeTadoClient{}, "1.0") }()

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

var _ tadoClient = &fakeTadoClient{}

type fakeTadoClient struct{}

func (f fakeTadoClient) GetWeatherInfo(_ context.Context) (tado.WeatherInfo, error) {
	return tado.WeatherInfo{
		OutsideTemperature: tado.Temperature{Celsius: 25},
		SolarIntensity:     tado.Percentage{Percentage: 80},
		WeatherState:       tado.Value{Value: "SUNNY"},
	}, nil
}

func (f fakeTadoClient) GetMobileDevices(_ context.Context) ([]tado.MobileDevice, error) {
	return []tado.MobileDevice{{
		ID:       1,
		Name:     "foo",
		Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
		Location: tado.MobileDeviceLocation{AtHome: true},
	}}, nil
}

func (f fakeTadoClient) GetZones(_ context.Context) (tado.Zones, error) {
	return tado.Zones{
		{
			ID:   1,
			Name: "foo",
		},
	}, nil
}

func (f fakeTadoClient) GetZoneInfo(_ context.Context, i int) (tado.ZoneInfo, error) {
	if i != 1 {
		panic("not implemented")
	}
	return tado.ZoneInfo{
		Setting: tado.ZonePowerSetting{
			Type:        "",
			Power:       "",
			Temperature: tado.Temperature{Celsius: 22},
		},
		ActivityDataPoints: tado.ZoneInfoActivityDataPoints{
			HeatingPower: tado.Percentage{Percentage: 25.0},
		},
		SensorDataPoints: tado.ZoneInfoSensorDataPoints{
			InsideTemperature: tado.Temperature{Celsius: 21},
			Humidity:          tado.Percentage{Percentage: 55},
		},
	}, nil
}

func (f fakeTadoClient) GetHomeState(_ context.Context) (homeState tado.HomeState, err error) {
	return tado.HomeState{Presence: "HOME"}, nil
}

func (f fakeTadoClient) DeleteZoneOverlay(_ context.Context, _ int) error {
	//TODO implement me
	panic("implement me")
}

func (f fakeTadoClient) SetZoneTemporaryOverlay(_ context.Context, _ int, _ float64, _ time.Duration) error {
	//TODO implement me
	panic("implement me")
}

func (f fakeTadoClient) SetHomeState(_ context.Context, _ bool) error {
	//TODO implement me
	panic("implement me")
}

func (f fakeTadoClient) UnsetHomeState(_ context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (f fakeTadoClient) SetZoneOverlay(_ context.Context, _ int, _ float64) error {
	//TODO implement me
	panic("implement me")
}
