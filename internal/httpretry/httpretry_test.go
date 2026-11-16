package httpretry_test

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/httpretry"
)

func TestAfterReadsSecondsAndDates(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	for v, want := range map[string]time.Duration{
		"":     0,
		"30":   30 * time.Second,
		"-5":   0,
		"soon": 0,
		now.Add(time.Minute).Format(http.TimeFormat):  time.Minute,
		now.Add(-time.Minute).Format(http.TimeFormat): 0,
	} {
		if got := httpretry.After(v, now); got != want {
			t.Errorf("After(%q) = %s, want %s", v, got, want)
		}
	}
}

func TestWithoutURLKeepsTheTokenOut(t *testing.T) {
	err := &url.Error{Op: "Post", URL: "https://api.telegram.org/botSECRET/sendMessage", Err: errors.New("connection refused")}
	got := httpretry.WithoutURL(err).Error()
	if strings.Contains(got, "SECRET") || got != "connection refused" {
		t.Errorf("WithoutURL = %q", got)
	}
}
