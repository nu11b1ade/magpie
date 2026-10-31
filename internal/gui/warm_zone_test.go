package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/settings"
)

func TestSettingsWarmZone(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("TZ", "UTC")
	if err := settings.Save(settings.Settings{WarmZone: "Asia/Shanghai", NoAutoUpdate: true}); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, nil)
	for _, c := range []struct {
		body, zone string
		status     int
	}{
		{`{"theme":"dark","noAutoUpdate":true}`, "Asia/Shanghai", http.StatusOK},
		{`{"warmZone":" America/New_York ","noAutoUpdate":true}`, "America/New_York", http.StatusOK},
		{`{"warmZone":"Asia/NotAZone"}`, "America/New_York", http.StatusBadRequest},
		{`{"warmZone":"","noAutoUpdate":true}`, "", http.StatusOK},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/settings", strings.NewReader(c.body)))
		if rec.Code != c.status {
			t.Fatalf("save %s: status %d, want %d: %s", c.body, rec.Code, c.status, rec.Body)
		}
		if got := settings.Load().WarmZone; got != c.zone {
			t.Fatalf("save %s: zone %q, want %q", c.body, got, c.zone)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/settings", nil))
	var got struct {
		LocalZone string `json:"localZone"`
		WarmZone  string `json:"warmZone"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || got.LocalZone != "UTC" || got.WarmZone != "" {
		t.Fatalf("default zone: status %d, local %q, warm-up %q", rec.Code, got.LocalZone, got.WarmZone)
	}
}
