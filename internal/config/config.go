package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds the application configuration.
type Config struct {
	Provider string        `toml:"provider"`
	BaseURL  string        `toml:"base_url"`
	Model    string        `toml:"model"`
	APIKey   string        `toml:"api_key"`
	Lang     string        `toml:"lang"`
	Timeout  time.Duration `toml:"timeout"`
}

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
		Lang:     "tr",
		Timeout:  30 * time.Second,
	}
}

// LoadConfig loads configuration from file, environment, and flags.
// Precedence: flags > env vars > config file > defaults.
func LoadConfig(flagProvider, flagBaseURL, flagModel, flagAPIKey, flagLang string, flagTimeout time.Duration, flagConfigFile string) (*Config, error) {
	cfg := DefaultConfig()

	// 1. Load from config file
	configPath := flagConfigFile
	if configPath == "" {
		var err error
		configPath, err = ConfigFilePath()
		if err != nil {
			return nil, fmt.Errorf("config path: %w", err)
		}
	}
	if _, err := os.Stat(configPath); err == nil {
		if err := parseTOMLFile(configPath, cfg); err != nil {
			return nil, fmt.Errorf("config parse: %w", err)
		}
	}

	// 2. NIYE_* env vars
	if v := os.Getenv("NIYE_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("NIYE_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("NIYE_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("NIYE_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("NIYE_LANG"); v != "" {
		cfg.Lang = v
	}
	if v := os.Getenv("NIYE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
		}
	}

	// 3. CLI flags (highest precedence)
	if flagProvider != "" {
		cfg.Provider = flagProvider
	}
	if flagBaseURL != "" {
		cfg.BaseURL = flagBaseURL
	}
	if flagModel != "" {
		cfg.Model = flagModel
	}
	if flagAPIKey != "" {
		cfg.APIKey = flagAPIKey
	}
	if flagLang != "" {
		cfg.Lang = flagLang
	}
	if flagTimeout != 0 {
		cfg.Timeout = flagTimeout
	}

	// 4. Environment variable fallbacks (after provider is determined)
	if cfg.APIKey == "" {
		if cfg.Provider == "gemini" {
			if k := os.Getenv("GEMINI_API_KEY"); k != "" {
				cfg.APIKey = k
			}
		} else {
			if k := os.Getenv("OPENAI_API_KEY"); k != "" {
				cfg.APIKey = k
			}
		}
	}

	// Validate
	if cfg.Provider != "openai" && cfg.Provider != "gemini" {
		return nil, fmt.Errorf("geçersiz provider: %s (openai|gemini)", cfg.Provider)
	}
	if cfg.Lang != "tr" && cfg.Lang != "en" {
		return nil, fmt.Errorf("geçersiz dil: %s (tr|en)", cfg.Lang)
	}
	if cfg.Model == "" {
		return nil, errors.New("model boş olamaz")
	}

	return cfg, nil
}

// SaveConfig writes the config to the default config file with mode 0600.
func SaveConfig(cfg *Config) error {
	return SaveConfigToPath(cfg, "")
}

// SaveConfigToPath writes the config to the given path (or default if empty) with mode 0600.
func SaveConfigToPath(cfg *Config, customPath string) error {
	path := customPath
	if path == "" {
		var err error
		path, err = ConfigFilePath()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := marshalTOML(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ConfigDir returns the directory holding the config file.
// Respects XDG_CONFIG_HOME, falls back to ~/.config/niye.
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "niye"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "niye"), nil
}

// ConfigFilePath returns the path to the config file.
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// parseTOMLFile parses a minimal TOML file into the config struct.
func parseTOMLFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return parseTOML(string(data), cfg)
}

// parseTOML parses a minimal TOML string into the config struct.
// Supports: key = "value", key = 123, key = true, # comments.
func parseTOML(input string, cfg *Config) error {
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Remove inline comments
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Unquote string values
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}

		switch key {
		case "provider":
			cfg.Provider = val
		case "base_url":
			cfg.BaseURL = val
		case "model":
			cfg.Model = val
		case "api_key":
			cfg.APIKey = val
		case "lang":
			cfg.Lang = val
		case "timeout":
			if d, err := time.ParseDuration(val); err == nil {
				cfg.Timeout = d
			}
		}
	}
	return nil
}

// marshalTOML converts the config to TOML format.
func marshalTOML(cfg *Config) ([]byte, error) {
	var b strings.Builder
	b.WriteString("# niye config\n")
	b.WriteString(fmt.Sprintf("provider = \"%s\"\n", escapeTOML(cfg.Provider)))
	b.WriteString(fmt.Sprintf("base_url = \"%s\"\n", escapeTOML(cfg.BaseURL)))
	b.WriteString(fmt.Sprintf("model = \"%s\"\n", escapeTOML(cfg.Model)))
	if cfg.APIKey != "" {
		b.WriteString(fmt.Sprintf("api_key = \"%s\"\n", escapeTOML(cfg.APIKey)))
	}
	b.WriteString(fmt.Sprintf("lang = \"%s\"\n", escapeTOML(cfg.Lang)))
	b.WriteString(fmt.Sprintf("timeout = \"%s\"\n", cfg.Timeout.String()))
	return []byte(b.String()), nil
}

func escapeTOML(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// ValidateConfig checks if the config has required fields.
func ValidateConfig(cfg *Config) error {
	if cfg.Provider != "openai" && cfg.Provider != "gemini" {
		return fmt.Errorf("geçersiz provider: %s (openai|gemini)", cfg.Provider)
	}
	if cfg.Lang != "tr" && cfg.Lang != "en" {
		return fmt.Errorf("geçersiz dil: %s (tr|en)", cfg.Lang)
	}
	if cfg.Model == "" {
		return errors.New("model boş olamaz")
	}
	if cfg.APIKey == "" {
		return errors.New("API anahtarı bulunamadı. NIYE_API_KEY ayarla veya \"niye init\" çalıştır")
	}
	return nil
}
