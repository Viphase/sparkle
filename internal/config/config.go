package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/viphase/sparkle/internal/workspace"
)

const FileName = "config.toml"

// Config holds user preferences loaded from .sparkle/config.toml.
type Config struct {
	Theme          string
	WordsThreshold int
}

func Defaults() Config {
	return Config{
		Theme:          "pastel-dark",
		WordsThreshold: 10,
	}
}

// Path returns the config path inside a workspace root.
func Path(root string) string {
	return filepath.Join(root, workspace.MetaDirName, FileName)
}

// Ensure loads the workspace config, creating a default config only when the
// file is missing. Existing files are never rewritten by this path.
func Ensure(root string) (Config, error) {
	path := Path(root)
	if _, err := os.Stat(path); err == nil {
		return Load(root)
	} else if !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("stat config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Config{}, fmt.Errorf("mkdir config dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return Load(root)
		}
		return Config{}, fmt.Errorf("create config: %w", err)
	}
	if _, err := f.Write(defaultConfigBytes()); err != nil {
		_ = f.Close()
		return Config{}, fmt.Errorf("write config: %w", err)
	}
	if err := f.Close(); err != nil {
		return Config{}, fmt.Errorf("close config: %w", err)
	}
	return Defaults(), nil
}

// Load reads .sparkle/config.toml. Missing files return Defaults without
// creating anything; use Ensure for first-run bootstrap.
func Load(root string) (Config, error) {
	cfg := Defaults()
	raw, err := os.ReadFile(Path(root))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if err := parseInto(&cfg, string(raw)); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func parseInto(cfg *Config, raw string) error {
	for lineNo, line := range strings.Split(raw, "\n") {
		line = stripComment(line)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("parse config line %d: expected key = value", lineNo+1)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "theme":
			s, err := parseString(value)
			if err != nil {
				return fmt.Errorf("parse config line %d: %w", lineNo+1, err)
			}
			cfg.Theme = s
		case "words_threshold":
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("parse config line %d: words_threshold must be an integer", lineNo+1)
			}
			if n < 1 {
				return fmt.Errorf("parse config line %d: words_threshold must be positive", lineNo+1)
			}
			cfg.WordsThreshold = n
		default:
			// Preserve forward compatibility: future config keys should not
			// make an older binary refuse to boot.
			continue
		}
	}
	if strings.TrimSpace(cfg.Theme) == "" {
		cfg.Theme = Defaults().Theme
	}
	return nil
}

func parseString(value string) (string, error) {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		s, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("invalid quoted string")
		}
		return strings.TrimSpace(s), nil
	}
	if strings.ContainsAny(value, " \t") {
		return "", fmt.Errorf("strings with whitespace must be quoted")
	}
	return strings.TrimSpace(value), nil
}

func stripComment(line string) string {
	inQuote := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inQuote {
			escaped = true
			continue
		}
		if r == '"' {
			inQuote = !inQuote
			continue
		}
		if r == '#' && !inQuote {
			return line[:i]
		}
	}
	return line
}

// Save writes cfg to the workspace config file, overwriting any existing file.
func Save(root string, cfg Config) error {
	path := Path(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	if err := os.WriteFile(path, marshalConfig(cfg), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func marshalConfig(cfg Config) []byte {
	return []byte(fmt.Sprintf("# Sparkle workspace preferences\ntheme = %q\nwords_threshold = %d\n",
		cfg.Theme,
		cfg.WordsThreshold,
	))
}

func defaultConfigBytes() []byte {
	return marshalConfig(Defaults())
}
