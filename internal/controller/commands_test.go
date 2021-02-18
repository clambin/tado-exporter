package controller

import (
	"github.com/clambin/tado-exporter/internal/tadoproxy"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestController_doRooms(t *testing.T) {
	control := Controller{
		proxy: tadoproxy.Proxy{
			API: &mockapi.MockAPI{},
		},
	}

	err := control.proxy.Refresh()
	assert.Nil(t, err)

	output := control.doUsers()
	assert.Len(t, output, 2)
	assert.Equal(t, "bar: away", output[0])
	assert.Equal(t, "foo: home", output[1])
}

func TestController_doUsers(t *testing.T) {
	control := Controller{
		proxy: tadoproxy.Proxy{
			API: &mockapi.MockAPI{},
		},
	}
	err := control.proxy.Refresh()
	assert.Nil(t, err)

	output := control.doUsers()
	assert.Len(t, output, 2)
	assert.Equal(t, "bar: away", output[0])
	assert.Equal(t, "foo: home", output[1])
}
