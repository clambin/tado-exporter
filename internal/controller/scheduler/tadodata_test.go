package scheduler_test

import (
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testCase struct {
	searchID   int
	searchName string
	result     bool
	realID     int
	realName   string
}

var data = scheduler.TadoData{
	Zone: map[int]tado.Zone{
		1: {ID: 1, Name: "foo"},
		2: {ID: 2, Name: "bar"},
	},
	MobileDevice: map[int]tado.MobileDevice{
		1: {ID: 1, Name: "foo"},
		2: {ID: 2, Name: "bar"},
	},
}

func TestTadoData_LookupMobileDevice(t *testing.T) {
	for _, test := range []testCase{
		{searchID: 1, searchName: "", result: true, realID: 1, realName: "foo"},
		{searchID: 0, searchName: "foo", result: true, realID: 1, realName: "foo"},
		{searchID: 2, searchName: "", result: true, realID: 2, realName: "bar"},
		{searchID: 0, searchName: "bar", result: true, realID: 2, realName: "bar"},
		{searchID: 3, searchName: "", result: false},
	} {
		device := data.LookupMobileDevice(test.searchID, test.searchName)
		if test.result {
			assert.NotNil(t, device)
			assert.Equal(t, test.realID, device.ID)
			assert.Equal(t, test.realName, device.Name)
		} else {
			assert.Nil(t, device)
		}
	}
}

func TestTadoData_LookupZone(t *testing.T) {
	for _, test := range []testCase{
		{searchID: 1, searchName: "", result: true, realID: 1, realName: "foo"},
		{searchID: 0, searchName: "foo", result: true, realID: 1, realName: "foo"},
		{searchID: 2, searchName: "", result: true, realID: 2, realName: "bar"},
		{searchID: 0, searchName: "bar", result: true, realID: 2, realName: "bar"},
		{searchID: 3, searchName: "", result: false},
	} {
		zone := data.LookupZone(test.searchID, test.searchName)
		if test.result {
			assert.NotNil(t, zone)
			assert.Equal(t, test.realID, zone.ID)
			assert.Equal(t, test.realName, zone.Name)
		} else {
			assert.Nil(t, zone)
		}
	}
}
