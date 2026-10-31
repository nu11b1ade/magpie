package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/settings"
)

// A UTC container starts the windows at Shanghai's six, including an
// account's own time, once for each start even when its usage read lags.
func TestWarmAtInZone(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := settings.Save(settings.Settings{WarmZone: "Asia/Shanghai"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "warmup.json")
	f := &fakeWarm{}
	w := f.warmer(path)
	w.ownAt = func() map[string]string { return map[string]string{"b@example.com": "09:00"} }
	for _, c := range []struct{ now, sent string }{
		{"2026-09-27T21:59:00Z", ""},
		{"2026-09-27T22:00:00Z", "a@example.com"},
		{"2026-09-27T22:05:00Z", ""},
		{"2026-09-28T01:00:00Z", "b@example.com"},
		{"2026-09-28T22:00:00Z", "a@example.com"},
	} {
		f.now, _ = time.Parse(time.RFC3339, c.now)
		f.sent = nil
		f.ws = map[string][]QuotaWindow{
			"a@example.com": {{Name: "5 hours", Span: fiveHours}},
			"b@example.com": {{Name: "5 hours", Span: fiveHours}},
		}
		w.warmNow(t.Context(), "", []string{"06:00"})
		if got := strings.Join(f.sent, ","); got != c.sent {
			t.Fatalf("at %s sent to %q, want %q", c.now, got, c.sent)
		}
	}
}

func TestWarmLookInZone(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := settings.Save(settings.Settings{WarmZone: "Asia/Shanghai"}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 27, 22, 0, 0, 0, time.UTC)
	last := now.Add(-time.Minute)
	if !warmLook("", []string{"06:00"}, nil, false, last, now) {
		t.Fatal("the loop missed Shanghai's six on the UTC clock")
	}
	if !warmLook("", nil, map[string]string{"a@example.com": "06:00"}, false, last, now) {
		t.Fatal("the loop missed an account's own Shanghai time")
	}
}

func TestWarmAtDefaultClock(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	f := &fakeWarm{now: at8(28, 6, 0), ws: map[string][]QuotaWindow{
		"a@example.com": {{Name: "5 hours", Span: fiveHours}},
	}}
	f.runAt(t, filepath.Join(t.TempDir(), "warmup.json"), "", "06:00")
	if len(f.sent) != 1 {
		t.Fatal("an unset zone did not use the instant's own clock")
	}
}

func TestWarmAtUnknownZone(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := settings.Save(settings.Settings{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings.Path(), []byte(`{"warmZone":"Asia/NotAZone"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	now := at8(28, 6, 0)
	f := &fakeWarm{now: now, ws: map[string][]QuotaWindow{
		"a@example.com": {{Name: "5 hours", Span: fiveHours}},
	}}
	f.runAt(t, filepath.Join(t.TempDir(), "warmup.json"), "", "06:00")
	if len(f.sent) != 0 || warmLook("", []string{"06:00"}, nil, true, time.Time{}, now) {
		t.Fatal("an unknown warm-up zone fell back to another clock")
	}
}

func TestWarmResetWaitsInZone(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := settings.Save(settings.Settings{WarmZone: "Asia/Shanghai"}); err != nil {
		t.Fatal(err)
	}
	for _, which := range []string{"week", "all"} {
		f := &fakeWarm{now: time.Date(2026, 9, 27, 18, 0, 0, 0, time.UTC), ws: map[string][]QuotaWindow{
			"a@example.com": {{Name: "5 hours", Span: fiveHours}, {Name: "7 days", Span: week}},
		}}
		path := filepath.Join(t.TempDir(), "warmup.json")
		if rs := f.runAt(t, path, which, "06:00"); len(rs) != 0 {
			t.Fatalf("%s: a window started before Shanghai's six: %+v", which, rs)
		}
		f.now = time.Date(2026, 9, 27, 22, 0, 0, 0, time.UTC)
		if rs := f.runAt(t, path, which, "06:00"); len(rs) != 1 || len(rs[0].Windows) != 2 {
			t.Fatalf("%s: Shanghai's six did not start both windows: %+v", which, rs)
		}
	}
}
