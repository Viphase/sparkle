package config

// Config holds user preferences loaded from .sparkle/config.toml.
// The TOML loader lands when storage is wired up; until then Defaults() is the only entry point.
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
