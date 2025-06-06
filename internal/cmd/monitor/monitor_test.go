package monitor

import (
	"github.com/clambin/tado-exporter/internal/cmd/monitor/mocks"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado/v2"
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
		want    controller.Configuration
	}{
		{
			name: "valid",
			content: `
zones:
    bathroom:
        - name: limitOverlay
          script:
            packaged: limitoverlay.lua
`,
			wantErr: assert.NoError,
			want: controller.Configuration{
				Zones: map[string][]rules.RuleConfiguration{
					"bathroom": {{
						Name:   "limitOverlay",
						Script: rules.ScriptConfig{Packaged: "limitoverlay.lua"},
					}},
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
			f, err := os.CreateTemp("", "")
			require.NoError(t, err)

			if tt.content != "" {
				_, err := f.Write([]byte(tt.content))
				require.NoError(t, err)
				_ = f.Close()
				t.Cleanup(func() { _ = os.Remove(f.Name()) })
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

	ctx := t.Context()
	client := mocks.NewTadoClient(t)
	client.EXPECT().
		GetMeWithResponse(ctx).
		Return(&tado.GetMeResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200:      &tado.User{Homes: &[]tado.HomeBase{{Id: oapi.VarP[tado.HomeId](1), Name: oapi.VarP("home")}}},
		}, nil)
	client.EXPECT().
		GetHomeStateWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.GetHomeStateResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200:      &tado.HomeState{Presence: oapi.VarP[tado.HomePresence](tado.HOME)},
		}, nil)
	client.EXPECT().
		GetZonesWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.GetZonesResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &[]tado.Zone{},
		}, nil)
	client.EXPECT().
		GetMobileDevicesWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.GetMobileDevicesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200:      &[]tado.MobileDevice{},
		}, nil)
	client.EXPECT().
		GetWeatherWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.GetWeatherResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &tado.Weather{
				OutsideTemperature: &tado.TemperatureDataPoint{Celsius: oapi.VarP[float32](18.5)},
				SolarIntensity:     &tado.PercentageDataPoint{Percentage: oapi.VarP[float32](25.0)},
				WeatherState:       &tado.WeatherStateDataPoint{Value: oapi.VarP(tado.RAIN)},
			},
		}, nil)

	l := slog.New(slog.DiscardHandler)
	go func() { assert.NoError(t, run(ctx, l, v, r, client, nil)) }()

	assert.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:9090/metrics")
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return err == nil && resp.StatusCode == http.StatusOK
	}, time.Second, 50*time.Millisecond)

	assert.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:8080/health")
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return err == nil && resp.StatusCode == http.StatusOK
	}, time.Second, 50*time.Millisecond)
}
