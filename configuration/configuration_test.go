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
		env      EnvVars
	}{
		{filename: "testdata/complete.yaml", pass: true},
		{filename: "testdata/token_envvar.yaml", pass: true, env: EnvVars{"TADO_TOKEN": "1234"}},
		{filename: "testdata/invalid.yaml", pass: false},
		{filename: "not-a-file", pass: false},
	}

	for _, tt := range testCases {
		err := tt.env.Set()
		require.NoError(t, err)

		cfg, err := configuration.LoadConfigurationFile(tt.filename)
		if tt.pass == false {
			assert.Error(t, err, tt.filename)
			continue
		}
		require.NoError(t, err, tt.filename)

		var body, golden []byte
		body, err = yaml.Marshal(cfg)
		require.NoError(t, err, tt.filename)

		gp := filepath.Join("testdata", t.Name()+"-"+slug.Make(tt.filename)+".golden")
		if *update {
			err = os.WriteFile(gp, body, 0644)
			require.NoError(t, err, tt.filename)
		}

		golden, err = os.ReadFile(gp)
		require.NoError(t, err, tt.filename)
		assert.Equal(t, string(golden), string(body), tt.filename)

		err = tt.env.Clear()
		require.NoError(t, err)
	}
}

type EnvVars map[string]string

func (e EnvVars) Set() error {
	for key, value := range e {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

func (e EnvVars) Clear() error {
	for key := range e {
		if err := os.Unsetenv(key); err != nil {
			return err
		}
	}
	return nil
}
