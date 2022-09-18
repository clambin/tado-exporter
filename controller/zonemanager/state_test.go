package zonemanager

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestState(t *testing.T) {
	m := New(nil, nil, nil, config)

	for _, tt := range testCases {
		t.Log(tt.name)
		current, next := m.getNextState(tt.update)
		assert.Equal(t, tt.current, current, tt.name)
		assert.Equal(t, tt.next, next, tt.name)
	}
}
