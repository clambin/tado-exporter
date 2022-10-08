package controller_test

/*
var (
	cfg = &configuration.ControllerConfiguration{
		Enabled: true,
		ZoneConfig: []configuration.ZoneConfig{{
			ZoneID:   1,
			ZoneName: "foo",
			AutoAway: configuration.ZoneAutoAway{
				Enabled: true,
				Delay:   2 * time.Hour,
				Users: []configuration.ZoneUser{
					{MobileDeviceID: 10, MobileDeviceName: "foo"},
				},
			},
			LimitOverlay: configuration.ZoneLimitOverlay{Enabled: true, Delay: time.Hour},
			NightTime:    configuration.ZoneNightTime{Enabled: true, Time: configuration.ZoneNightTimeTimestamp{Hour: 23, Minutes: 30}},
		}},
	}
)

	func TestController_Run(t *testing.T) {
		//log.SetLevel(log.DebugLevel)
		a := &mocks.API{}
		prepareMockAPI(a)

		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}

		p := poller.New(a)
		wg.Add(1)
		go func() {
			p.Run(ctx, time.Second)
			wg.Done()
		}()

		b := &mocks2.SlackBot{}
		b.On("RegisterCallback", mock.AnythingOfType("string"), mock.AnythingOfType("slackbot.CommandFunc")).Return(nil)

		c := controller.New(a, cfg, b, p)
		assert.NotNil(t, c)

		wg.Add(1)
		go func() {
			c.Run(ctx, time.Minute)
			wg.Done()
		}()

		time.Sleep(5 * time.Second)

		cancel()
		wg.Wait()

		mock.AssertExpectationsForObjects(t, a, b)
	}

func prepareMockAPI(api *mocks.API) {
	api.
		On("GetMobileDevices", mock.Anything).
		Return([]tado.MobileDevice{
			{
				ID:       10,
				Name:     "foo",
				Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
				Location: tado.MobileDeviceLocation{AtHome: true},
			},
			{
				ID:       11,
				Name:     "bar",
				Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
				Location: tado.MobileDeviceLocation{AtHome: false},
			}}, nil)
	api.
		On("GetWeatherInfo", mock.Anything).
		Return(tado.WeatherInfo{
			OutsideTemperature: tado.Temperature{Celsius: 3.4},
			SolarIntensity:     tado.Percentage{Percentage: 13.3},
			WeatherState:       tado.Value{Value: "CLOUDY_MOSTLY"},
		}, nil)
	api.On("GetZones", mock.Anything).
		Return([]tado.Zone{
			{ID: 1, Name: "foo"},
			{ID: 2, Name: "bar"},
		}, nil)
	api.
		On("GetZoneInfo", mock.Anything, 1).
		Return(tado.ZoneInfo{
			Setting: tado.ZoneInfoSetting{
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 18.5},
			},
		}, nil)
	api.
		On("GetZoneInfo", mock.Anything, 2).
		Return(tado.ZoneInfo{
			Setting: tado.ZoneInfoSetting{
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 15.0},
			},
			Overlay: tado.ZoneInfoOverlay{
				Type: "MANUAL",
				Setting: tado.ZoneInfoOverlaySetting{
					Type:        "HEATING",
					Power:       "OFF",
					Temperature: tado.Temperature{Celsius: 5.0},
				},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			},
		}, nil)
}
*/
