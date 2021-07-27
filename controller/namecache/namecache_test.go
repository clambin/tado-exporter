package namecache_test

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/namecache"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCache_GetName(t *testing.T) {
	cache := namecache.New()
	cache.Update(&poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
		Zones: map[int]tado.Zone{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
	})

	name, ok := cache.GetZoneName(1)
	assert.True(t, ok)
	assert.Equal(t, "foo", name)

	name, ok = cache.GetZoneName(3)
	assert.False(t, ok)

	name, ok = cache.GetUserName(2)
	assert.True(t, ok)
	assert.Equal(t, "bar", name)

	name, ok = cache.GetUserName(3)
	assert.False(t, ok)

	var id int
	id, name, ok = cache.LookupZone(1, "")
	assert.True(t, ok)
	assert.Equal(t, 1, id)
	assert.Equal(t, "foo", name)

	id, name, ok = cache.LookupZone(0, "bar")
	assert.True(t, ok)
	assert.Equal(t, 2, id)
	assert.Equal(t, "bar", name)

	id, name, ok = cache.LookupZone(0, "snafu")
	assert.False(t, ok)

	id, name, ok = cache.LookupUser(1, "")
	assert.True(t, ok)
	assert.Equal(t, 1, id)
	assert.Equal(t, "foo", name)

	id, name, ok = cache.LookupUser(0, "bar")
	assert.True(t, ok)
	assert.Equal(t, 2, id)
	assert.Equal(t, "bar", name)

	id, name, ok = cache.LookupUser(0, "snafu")
	assert.False(t, ok)
}

func BenchmarkCache_Update(b *testing.B) {
	cache := namecache.New()
	update := &poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
		Zones: map[int]tado.Zone{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
	}

	for i := 0; i < 1000000; i++ {
		cache.Update(update)
	}
}
