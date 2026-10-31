package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

func isolateSettings(t *testing.T) {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
}

// A named warm-up zone is saved, shown, and kept when an older page omits
// it. A name magpie doesn't know is refused and leaves the saved one.
// Clearing it is this computer's clock again.
func TestWarmZoneRoundTrip(t *testing.T) {
	isolateSettings(t)
	if err := settings.Save(settings.Settings{Theme: "light", WarmZone: "Asia/Shanghai"}); err != nil {
		t.Fatal(err)
	}
	post := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		Handler(nil, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/settings", strings.NewReader(body)))
		return rec
	}
	rec := httptest.NewRecorder()
	Handler(nil, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("get %d %s", rec.Code, rec.Body)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["warmZone"] != "Asia/Shanghai" {
		t.Fatalf("warmZone %v", got["warmZone"])
	}
	if z, _ := got["localZone"].(string); z != provider.LocalZone() {
		t.Fatalf("localZone %q, want %q", z, provider.LocalZone())
	}
	if rec := post(`{"theme":"dark"}`); rec.Code != http.StatusOK || settings.Load().WarmZone != "Asia/Shanghai" || settings.Load().Theme != "dark" {
		t.Fatalf("omitted zone: %d %s saved %q", rec.Code, rec.Body, settings.Load().WarmZone)
	}
	if rec := post(`{"theme":"dark","warmZone":"UTC"}`); rec.Code != http.StatusOK || settings.Load().WarmZone != "UTC" {
		t.Fatalf("set UTC: %d %s saved %q", rec.Code, rec.Body, settings.Load().WarmZone)
	}
	if rec := post(`{"theme":"dark","warmZone":"Not/AZone"}`); rec.Code == http.StatusOK || settings.Load().WarmZone != "UTC" {
		t.Fatalf("bad zone accepted: %d %s saved %q", rec.Code, rec.Body, settings.Load().WarmZone)
	}
	if rec := post(`{"theme":"dark","warmZone":""}`); rec.Code != http.StatusOK || settings.Load().WarmZone != "" {
		t.Fatalf("cleared: %d %s saved %q", rec.Code, rec.Body, settings.Load().WarmZone)
	}
}
