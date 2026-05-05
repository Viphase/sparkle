package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadMissingReturnsDefaults(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg != Defaults() {
		t.Fatalf("cfg=%+v, want defaults %+v", cfg, Defaults())
	}
}

func TestEnsureCreatesDefaultConfigWithoutOverwriting(t *testing.T) {
	root := t.TempDir()
	cfg, err := Ensure(root)
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	if cfg != Defaults() {
		t.Fatalf("cfg=%+v, want defaults %+v", cfg, Defaults())
	}
	raw, err := os.ReadFile(Path(root))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(raw), `theme = "pastel-dark"`) {
		t.Fatalf("default config missing theme: %s", raw)
	}

	custom := []byte("theme = \"nova\"\nwords_threshold = 25\n")
	if err := os.WriteFile(Path(root), custom, 0o644); err != nil {
		t.Fatalf("write custom config: %v", err)
	}
	cfg, err = Ensure(root)
	if err != nil {
		t.Fatalf("Ensure custom returned error: %v", err)
	}
	if cfg.Theme != "nova" || cfg.WordsThreshold != 25 {
		t.Fatalf("cfg=%+v, want custom values", cfg)
	}
	raw, err = os.ReadFile(Path(root))
	if err != nil {
		t.Fatalf("read custom config: %v", err)
	}
	if string(raw) != string(custom) {
		t.Fatalf("Ensure rewrote existing config: %q", raw)
	}
}

func TestLoadParsesCommentsAndUnknownKeys(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	raw := []byte(`
# comment
theme = "pastel-light" # inline comment
future_key = "ignored"
words_threshold = 7
`)
	if err := os.WriteFile(Path(root), raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Theme != "pastel-light" || cfg.WordsThreshold != 7 {
		t.Fatalf("cfg=%+v, want parsed values", cfg)
	}
}

// TestSectionedRoundTrip writes a sectioned config (the v2 format) and reads
// it back; all fields including the new tracking durations survive.
func TestSectionedRoundTrip(t *testing.T) {
	root := t.TempDir()
	original := Config{
		Theme:           "nova",
		WordsThreshold:  42,
		MouseEnabled:    false,
		AnthropicAPIKey: "sk-ant-test",
		AIModel:         "claude-sonnet-4-6",
		ActiveSkill:     "cli-tool",
		TouchWindow:     14 * 24 * time.Hour,
		SessionIdle:     30 * time.Minute,
		StreakGrace:     48 * time.Hour,
	}
	if err := Save(root, original); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, _ := os.ReadFile(Path(root))
	if !strings.Contains(string(raw), "[appearance]") || !strings.Contains(string(raw), "[tracking]") {
		t.Fatalf("expected sectioned output, got:\n%s", raw)
	}
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != original {
		t.Fatalf("round-trip mismatch:\n got=%+v\nwant=%+v", loaded, original)
	}
}

// TestLoadV1FlatConfig confirms a v1 flat config file (no section headers,
// no tracking durations) still loads cleanly with new fields defaulted.
func TestLoadV1FlatConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	v1 := []byte(`# v1 flat config
theme = "pastel-light"
words_threshold = 12
mouse_enabled = false
ai_model = "claude-haiku-4-5"
active_skill = "web-api"
`)
	if err := os.WriteFile(Path(root), v1, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load v1: %v", err)
	}
	if cfg.Theme != "pastel-light" || cfg.WordsThreshold != 12 || cfg.MouseEnabled != false {
		t.Fatalf("v1 fields not parsed: %+v", cfg)
	}
	if cfg.ActiveSkill != "web-api" {
		t.Fatalf("active_skill not parsed: %+v", cfg)
	}
	d := Defaults()
	if cfg.TouchWindow != d.TouchWindow || cfg.SessionIdle != d.SessionIdle || cfg.StreakGrace != d.StreakGrace {
		t.Fatalf("new tracking fields not defaulted: %+v", cfg)
	}
}

func TestLoadParsesDurations(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	raw := []byte(`[tracking]
touch_window = "3h"
session_idle = "15m"
streak_grace = "24h"
`)
	if err := os.WriteFile(Path(root), raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TouchWindow != 3*time.Hour || cfg.SessionIdle != 15*time.Minute || cfg.StreakGrace != 24*time.Hour {
		t.Fatalf("durations not parsed: %+v", cfg)
	}
}

func TestLoadRejectsInvalidThreshold(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(Path(root), []byte("words_threshold = 0\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("expected invalid threshold error")
	}
}
