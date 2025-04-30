package slacktools

import (
	"encoding/json"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

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

type postResponse struct {
	Ok      bool   `json:"ok"`
	Channel string `json:"channel"`
	Ts      string `json:"ts"`
	Message struct {
		Text string `json:"text"`
		Type string `json:"type"`
		Ts   string `json:"ts"`
	} `json:"message"`
}

func TestAttachment_Format(t *testing.T) {
	var h fakeServerHandler
	s := httptest.NewServer(&h)
	t.Cleanup(s.Close)

	c := slack.New("12345678", slack.OptionAPIURL(s.URL+"/"))

	attachment := Attachment{
		Header: "title",
		Body:   []string{"line 1", "line 2"},
	}

	_, err := c.PostEphemeral("channel", "UFAKE123", attachment.Format())
	require.NoError(t, err)
	want := url.Values{
		"blocks":  []string{`[{"type":"section","text":{"type":"mrkdwn","text":"*title*"},"fields":[{"type":"mrkdwn","text":"line 1"},{"type":"mrkdwn","text":"line 2"}]}]`},
		"channel": []string{"channel"},
		"token":   []string{"12345678"},
		"user":    []string{"UFAKE123"},
	}

	assert.Equal(t, want, h.lastReceived.Load().(url.Values))
}

var _ http.Handler = &fakeServerHandler{}

type fakeServerHandler struct {
	lastReceived atomic.Value
}

func (f *fakeServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}
	body, _ := io.ReadAll(r.Body)
	values, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	f.lastReceived.Store(values)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(postResponse{
		Ok:      true,
		Channel: "CFAKE123",
		Ts:      "1714220000.123456",
	})
}
