package slacktools

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAttachment_build(t *testing.T) {
	a := Attachment{
		Header: "header",
		Body: []string{
			"line 1",
			"line 2",
		},
	}

	b := a.build()
	assert.Equal(t, "*"+a.Header+"*", b.Text.Text)
	for i, l := range b.Fields {
		assert.Equal(t, a.Body[i], l.Text)
	}
}

func TestAttachment_IsZero(t *testing.T) {
	tests := []struct {
		name   string
		a      Attachment
		isZero assert.BoolAssertionFunc
	}{
		{
			name:   "empty",
			a:      Attachment{},
			isZero: assert.True,
		},
		{
			name:   "header only",
			a:      Attachment{Header: "title"},
			isZero: assert.False,
		},
		{
			name:   "body only",
			a:      Attachment{Body: []string{"body"}},
			isZero: assert.False,
		},
		{
			name:   "header and body",
			a:      Attachment{Header: "title", Body: []string{"body"}},
			isZero: assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.isZero(t, tt.a.IsZero())
		})
	}
}
