package configuration_test

import (
	"flag"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/gosimple/slug"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update .golden files")

func TestLoadConfiguration(t *testing.T) {
	testCases := []struct {
		filename string
		pass     bool
	}{
		{filename: "testdata/complete.yaml", pass: true},
		{filename: "testdata/token_envvar.yaml", pass: true},
		{filename: "testdata/invalid.yaml", pass: false},
		{filename: "not-a-file", pass: false},
	}

	_ = os.Setenv("TADO_TOKEN", "1234")

	for _, tt := range testCases {
		cfg, err := configuration.LoadConfigurationFile(tt.filename)
		if tt.pass == false {
			assert.Error(t, err, tt.filename)
			continue
		}
		require.NoError(t, err, tt.filename)

		var body, golden []byte
		body, err = yaml.Marshal(cfg)
		require.NoError(t, err, tt.filename)

		assert.Equal(t, "1234", cfg.Controller.TadoBot.Token.Value)

		gp := filepath.Join("testdata", t.Name()+"-"+slug.Make(tt.filename)+".golden")
		if *update {
			err = os.WriteFile(gp, body, 0644)
			require.NoError(t, err, tt.filename)
		}

		golden, err = os.ReadFile(gp)
		require.NoError(t, err, tt.filename)
		assert.Equal(t, string(golden), string(body), tt.filename)
	}
}
