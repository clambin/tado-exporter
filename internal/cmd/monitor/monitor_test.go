package monitor

import (
	"context"
	mocks2 "github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/cmd/monitor/mocks"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	client.EXPECT().GetMeWithResponse(ctx).
		Return(&tado.GetMeResponse{JSON200: &tado.User{Homes: &[]tado.HomeBase{{Id: oapi.VarP[tado.HomeId](1), Name: oapi.VarP("home")}}}}, nil)
	client.EXPECT().GetHomeStateWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.GetHomeStateResponse{JSON200: &tado.HomeState{Presence: oapi.VarP[tado.HomePresence](tado.HOME)}}, nil)
	client.EXPECT().GetZonesWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.GetZonesResponse{JSON200: &[]tado.Zone{}}, nil)
	client.EXPECT().GetMobileDevicesWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.GetMobileDevicesResponse{JSON200: &[]tado.MobileDevice{}}, nil)
	client.EXPECT().GetWeatherWithResponse(ctx, tado.HomeId(1)).
		Return(&tado.GetWeatherResponse{JSON200: &tado.Weather{
			OutsideTemperature: &tado.TemperatureDataPoint{Celsius: oapi.VarP[float32](18.5)},
			SolarIntensity:     &tado.PercentageDataPoint{Percentage: oapi.VarP[float32](25.0)},
			WeatherState:       &tado.WeatherStateDataPoint{Value: oapi.VarP(tado.RAIN)},
		}}, nil)
	slack := mocks2.NewSlackBot(t)
	slack.EXPECT().Add(mock.AnythingOfType("slackbot.Commands"))
	slack.EXPECT().Run(ctx).Return(nil)

	errCh := make(chan error)
	go func() { errCh <- run(ctx, l, v, r, client, slack) }()

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
