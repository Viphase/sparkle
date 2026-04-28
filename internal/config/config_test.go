package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
