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
