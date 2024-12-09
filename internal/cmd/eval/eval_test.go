package eval

import (
	"bytes"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_evalHomeRule(t *testing.T) {
	v := viper.New()
	v.Set("action-only", true)
	var output bytes.Buffer

	assert.NoError(t, evalHomeRule(&output, v)(nil, []string{"../../controller/rules/homerules/homeandaway.lua"}))

	const want = `INPUT                                                                                      CHANGE REASON                                   ACTION
home(overlay:false,home:false) user(home:true)                                             true   one or more users are home: user         setting home to HOME mode in 0s
home(overlay:false,home:true) user(home:false)                                             true   all users are away: user                 setting home to AWAY mode in 5m0s
home(overlay:true,home:false) user(home:true)                                              true   one or more users are home: user         setting home to HOME mode in 0s
home(overlay:true,home:true) user(home:false)                                              true   all users are away: user                 setting home to AWAY mode in 5m0s
`

	assert.Equal(t, want, output.String())
}

func Test_evalZoneRule(t *testing.T) {
	v := viper.New()
	v.Set("action-only", true)
	var output bytes.Buffer

	assert.NoError(t, evalZoneRule(&output, v)(nil, []string{"../../controller/rules/zonerules/limitoverlay.lua"}))

	const want = `INPUT                                                                                      CHANGE REASON                                   ACTION
home(overlay:false,home:false) zone(overlay:true,heating: true) user(home:false)           true   manual setting detected                  *zone*: switching heating to auto mode in 1h0m0s
home(overlay:false,home:false) zone(overlay:true,heating: true) user(home:true)            true   manual setting detected                  *zone*: switching heating to auto mode in 1h0m0s
home(overlay:false,home:true) zone(overlay:true,heating: true) user(home:false)            true   manual setting detected                  *zone*: switching heating to auto mode in 1h0m0s
home(overlay:false,home:true) zone(overlay:true,heating: true) user(home:true)             true   manual setting detected                  *zone*: switching heating to auto mode in 1h0m0s
home(overlay:true,home:false) zone(overlay:true,heating: true) user(home:false)            true   manual setting detected                  *zone*: switching heating to auto mode in 1h0m0s
home(overlay:true,home:false) zone(overlay:true,heating: true) user(home:true)             true   manual setting detected                  *zone*: switching heating to auto mode in 1h0m0s
home(overlay:true,home:true) zone(overlay:true,heating: true) user(home:false)             true   manual setting detected                  *zone*: switching heating to auto mode in 1h0m0s
home(overlay:true,home:true) zone(overlay:true,heating: true) user(home:true)              true   manual setting detected                  *zone*: switching heating to auto mode in 1h0m0s
`

	assert.Equal(t, want, output.String())
}
