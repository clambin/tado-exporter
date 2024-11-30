package tmp

import (
	"github.com/Shopify/go-lua"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDevices(t *testing.T) {
	input := devices{
		{"user A", true},
		{"user B", true},
		{"user C", false},
	}

	filtered := input.filter(set.New("user A", "user C"))
	assert.Equal(t, devices{
		{"user A", true},
		{"user C", false},
	}, filtered)

	l := lua.NewState()
	filtered.toLua(l)

	got := luart.TableToSlice(l, l.AbsIndex(-1))
	require.Len(t, got, 2)
}
