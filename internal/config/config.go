package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/viphase/sparkle/internal/workspace"
)

const FileName = "config.toml"

// Config holds user preferences loaded from .sparkle/config.toml.
type Config struct {
	Theme          string
	WordsThreshold int
	// MouseEnabled controls tea.EnableMouseCellMotion. Defaults to true.
	MouseEnabled bool
	// AnthropicAPIKey enables the real Anthropic provider. If empty the app
	// falls back to the local mock provider. Can also be supplied through the
	// ANTHROPIC_API_KEY environment variable; the env var wins over the file.
	AnthropicAPIKey string
	// AIModel is the Anthropic model used when the real provider is active.
	// Defaults to "claude-haiku-4-5".
	AIModel string
	// ActiveSkill selects an injectable prompt skill that specialises the AI
	// guide for a specific project type. Empty string means no specialisation.
	// Valid values match domain.Skill constants (e.g. "cli-tool", "web-api").
	ActiveSkill string
	// TouchWindow is the recency cutoff for a "recently touched" project.
	TouchWindow time.Duration
	// SessionIdle is the inactivity gap that ends a writing session.
	SessionIdle time.Duration
	// StreakGrace is how long a streak survives a missed day.
	StreakGrace time.Duration
}

func Defaults() Config {
	return Config{
		Theme:          "pastel-dark",
		WordsThreshold: 10,
		MouseEnabled:   true,
		AIModel:        "claude-haiku-4-5",
		TouchWindow:    7 * 24 * time.Hour,
		SessionIdle:    20 * time.Minute,
		StreakGrace:    36 * time.Hour,
	}
}

// ResolvedAPIKey returns the Anthropic API key, preferring the ANTHROPIC_API_KEY
// environment variable over the config-file value.
func (c Config) ResolvedAPIKey() string {
	if env := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")); env != "" {
		return env
	}
	return strings.TrimSpace(c.AnthropicAPIKey)
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
		// Skip TOML section headers ([section]) — sections are decorative; all
		// keys live in a flat namespace for back-compat with the v1 reader.
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
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
		case "mouse_enabled":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("parse config line %d: mouse_enabled must be true or false", lineNo+1)
			}
			cfg.MouseEnabled = b
		case "anthropic_api_key":
			s, err := parseString(value)
			if err != nil {
				return fmt.Errorf("parse config line %d: %w", lineNo+1, err)
			}
			cfg.AnthropicAPIKey = s
		case "ai_model":
			s, err := parseString(value)
			if err != nil {
				return fmt.Errorf("parse config line %d: %w", lineNo+1, err)
			}
			cfg.AIModel = s
		case "active_skill":
			s, err := parseString(value)
			if err != nil {
				return fmt.Errorf("parse config line %d: %w", lineNo+1, err)
			}
			cfg.ActiveSkill = s
		case "touch_window":
			d, err := parseDuration(value)
			if err != nil {
				return fmt.Errorf("parse config line %d: touch_window %w", lineNo+1, err)
			}
			cfg.TouchWindow = d
		case "session_idle":
			d, err := parseDuration(value)
			if err != nil {
				return fmt.Errorf("parse config line %d: session_idle %w", lineNo+1, err)
			}
			cfg.SessionIdle = d
		case "streak_grace":
			d, err := parseDuration(value)
			if err != nil {
				return fmt.Errorf("parse config line %d: streak_grace %w", lineNo+1, err)
			}
			cfg.StreakGrace = d
		default:
			// Preserve forward compatibility: future config keys should not
			// make an older binary refuse to boot.
			continue
		}
	}
	if strings.TrimSpace(cfg.Theme) == "" {
		cfg.Theme = Defaults().Theme
	}
	d := Defaults()
	if cfg.TouchWindow <= 0 {
		cfg.TouchWindow = d.TouchWindow
	}
	if cfg.SessionIdle <= 0 {
		cfg.SessionIdle = d.SessionIdle
	}
	if cfg.StreakGrace <= 0 {
		cfg.StreakGrace = d.StreakGrace
	}
	return nil
}

func parseDuration(value string) (time.Duration, error) {
	s, err := parseString(value)
	if err != nil {
		// duration value may be unquoted (e.g. session_idle = 20m).
		s = strings.TrimSpace(value)
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q (use Go duration syntax like 20m, 36h)", s)
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}
	return d, nil
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
	d := Defaults()
	if cfg.TouchWindow <= 0 {
		cfg.TouchWindow = d.TouchWindow
	}
	if cfg.SessionIdle <= 0 {
		cfg.SessionIdle = d.SessionIdle
	}
	if cfg.StreakGrace <= 0 {
		cfg.StreakGrace = d.StreakGrace
	}
	var b strings.Builder
	b.WriteString("# Sparkle workspace preferences\n\n")

	b.WriteString("[appearance]\n")
	b.WriteString(fmt.Sprintf("theme = %q\n\n", cfg.Theme))

	b.WriteString("[ai]\n")
	b.WriteString(fmt.Sprintf("ai_model = %q\n", cfg.AIModel))
	if cfg.ActiveSkill != "" {
		b.WriteString(fmt.Sprintf("active_skill = %q\n", cfg.ActiveSkill))
	}
	// anthropic_api_key is intentionally omitted from default writes to avoid
	// accidentally committing keys. Set it manually or use ANTHROPIC_API_KEY env.
	if cfg.AnthropicAPIKey != "" {
		b.WriteString(fmt.Sprintf("anthropic_api_key = %q\n", cfg.AnthropicAPIKey))
	}
	b.WriteString("\n")

	b.WriteString("[tracking]\n")
	b.WriteString(fmt.Sprintf("words_threshold = %d\n", cfg.WordsThreshold))
	b.WriteString(fmt.Sprintf("touch_window = %q\n", cfg.TouchWindow.String()))
	b.WriteString(fmt.Sprintf("session_idle = %q\n", cfg.SessionIdle.String()))
	b.WriteString(fmt.Sprintf("streak_grace = %q\n", cfg.StreakGrace.String()))
	b.WriteString("\n")

	b.WriteString("[mouse]\n")
	b.WriteString(fmt.Sprintf("mouse_enabled = %v\n", cfg.MouseEnabled))
	return []byte(b.String())
}

func defaultConfigBytes() []byte {
	return marshalConfig(Defaults())
}
