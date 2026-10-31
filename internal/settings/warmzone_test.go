package settings

import (
	"encoding/json"
	"testing"
)

func TestWarmZoneRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var s Settings
	if err := json.Unmarshal([]byte(`{"warmZone":"Asia/Shanghai","codexWarmAts":["06:00"]}`), &s); err != nil {
		t.Fatal(err)
	}
	if err := Save(s); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(Load())
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		WarmZone string `json:"warmZone"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.WarmZone != "Asia/Shanghai" {
		t.Fatalf("saved warm-up zone = %q, want Asia/Shanghai", got.WarmZone)
	}
}

func TestWarmZoneChecked(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Save(Settings{WarmZone: " Asia/Shanghai "}); err != nil {
		t.Fatal(err)
	}
	if got := Load().WarmZone; got != "Asia/Shanghai" {
		t.Fatalf("warm-up zone = %q, want trimmed IANA name", got)
	}
	if err := Save(Settings{WarmZone: "Asia/NotAZone"}); err == nil {
		t.Fatal("unknown warm-up zone accepted")
	}
	if got := Load().WarmZone; got != "Asia/Shanghai" {
		t.Fatalf("refused save replaced the warm-up zone with %q", got)
	}
	if err := Save(Settings{WarmZone: " "}); err != nil {
		t.Fatal(err)
	}
	if got := Load().WarmZone; got != "" {
		t.Fatalf("default warm-up zone = %q, want the local clock", got)
	}
}
