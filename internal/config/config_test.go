package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Provider != "openai" {
		t.Errorf("default provider = %q, want openai", cfg.Provider)
	}
	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("default base_url = %q", cfg.BaseURL)
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("default model = %q", cfg.Model)
	}
	if cfg.Lang != "tr" {
		t.Errorf("default lang = %q", cfg.Lang)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("default timeout = %v", cfg.Timeout)
	}
}

func TestParseTOML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Config
	}{
		{
			name: "basic config",
			input: `
provider = "gemini"
base_url = "https://custom.api"
model = "custom-model"
api_key = "test-key"
lang = "en"
timeout = "60s"
`,
			expected: Config{
				Provider: "gemini",
				BaseURL:  "https://custom.api",
				Model:    "custom-model",
				APIKey:   "test-key",
				Lang:     "en",
				Timeout:  60 * time.Second,
			},
		},
		{
			name: "with comments",
			input: `
# This is a comment
provider = "openai"
# Another comment
model = "gpt-4"
`,
			expected: Config{
				Provider: "openai",
				Model:    "gpt-4",
			},
		},
		{
			name: "inline comments",
			input: `
provider = "openai"  # comment
model = "gpt-4" # another
`,
			expected: Config{
				Provider: "openai",
				Model:    "gpt-4",
			},
		},
		{
			name: "quoted values",
			input: `
provider = 'openai'
model = "gpt-4"
`,
			expected: Config{
				Provider: "openai",
				Model:    "gpt-4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{} // zero value, not DefaultConfig
			if err := parseTOML(tt.input, cfg); err != nil {
				t.Fatalf("parseTOML error: %v", err)
			}
			if cfg.Provider != tt.expected.Provider {
				t.Errorf("provider = %q, want %q", cfg.Provider, tt.expected.Provider)
			}
			if cfg.BaseURL != tt.expected.BaseURL {
				t.Errorf("base_url = %q, want %q", cfg.BaseURL, tt.expected.BaseURL)
			}
			if cfg.Model != tt.expected.Model {
				t.Errorf("model = %q, want %q", cfg.Model, tt.expected.Model)
			}
			if cfg.APIKey != tt.expected.APIKey {
				t.Errorf("api_key = %q, want %q", cfg.APIKey, tt.expected.APIKey)
			}
			if cfg.Lang != tt.expected.Lang {
				t.Errorf("lang = %q, want %q", cfg.Lang, tt.expected.Lang)
			}
			if cfg.Timeout != tt.expected.Timeout {
				t.Errorf("timeout = %v, want %v", cfg.Timeout, tt.expected.Timeout)
			}
		})
	}
}

func TestMarshalTOML(t *testing.T) {
	cfg := &Config{
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
		APIKey:   "sk-test",
		Lang:     "tr",
		Timeout:  30 * time.Second,
	}

	data, err := marshalTOML(cfg)
	if err != nil {
		t.Fatalf("marshalTOML error: %v", err)
	}

	str := string(data)
	if !strings.Contains(str, `provider = "openai"`) {
		t.Errorf("missing provider in output: %s", str)
	}
	if !strings.Contains(str, `api_key = "sk-test"`) {
		t.Errorf("missing api_key in output: %s", str)
	}
}

func TestLoadConfigPrecedence(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
provider = "gemini"
base_url = "https://config.api"
model = "config-model"
api_key = "config-key"
lang = "en"
timeout = "10s"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Test 1: Config file only
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", "")
	cfg, err := LoadConfig("", "", "", "", "", 0, configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Provider != "gemini" {
		t.Errorf("provider = %q, want gemini", cfg.Provider)
	}
	if cfg.APIKey != "config-key" {
		t.Errorf("api_key = %q, want config-key", cfg.APIKey)
	}

	// Test 2: NIYE_* env vars override config file
	t.Setenv("NIYE_PROVIDER", "openai")
	t.Setenv("NIYE_API_KEY", "env-key")
	cfg, err = LoadConfig("", "", "", "", "", 0, configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Provider != "openai" {
		t.Errorf("provider = %q, want openai (from NIYE_PROVIDER)", cfg.Provider)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("api_key = %q, want env-key (from NIYE_API_KEY)", cfg.APIKey)
	}

	// Test 3: CLI flags override env vars
	cfg, err = LoadConfig("gemini", "https://flag.api", "flag-model", "flag-key", "en", 5*time.Second, configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Provider != "gemini" {
		t.Errorf("provider = %q, want gemini (from flag)", cfg.Provider)
	}
	if cfg.BaseURL != "https://flag.api" {
		t.Errorf("base_url = %q, want https://flag.api (from flag)", cfg.BaseURL)
	}
	if cfg.Model != "flag-model" {
		t.Errorf("model = %q, want flag-model (from flag)", cfg.Model)
	}
	if cfg.APIKey != "flag-key" {
		t.Errorf("api_key = %q, want flag-key (from flag)", cfg.APIKey)
	}
	if cfg.Lang != "en" {
		t.Errorf("lang = %q, want en (from flag)", cfg.Lang)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s (from flag)", cfg.Timeout)
	}
}

func TestLoadConfigFallbackKeys(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("OPENAI_API_KEY", "openai-fallback")
	t.Setenv("GEMINI_API_KEY", "gemini-fallback")

	// Test OpenAI fallback
	cfg, err := LoadConfig("openai", "", "", "", "", 0, "")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.APIKey != "openai-fallback" {
		t.Errorf("openai api_key = %q, want openai-fallback", cfg.APIKey)
	}

	// Test Gemini fallback
	cfg, err = LoadConfig("gemini", "", "", "", "", 0, "")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.APIKey != "gemini-fallback" {
		t.Errorf("gemini api_key = %q, want gemini-fallback", cfg.APIKey)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid openai",
			cfg: &Config{
				Provider: "openai",
				Model:    "gpt-4o-mini",
				APIKey:   "sk-test",
				Lang:     "tr",
			},
			wantErr: false,
		},
		{
			name: "valid gemini",
			cfg: &Config{
				Provider: "gemini",
				Model:    "gemini-1.5-flash",
				APIKey:   "gemini-key",
				Lang:     "en",
			},
			wantErr: false,
		},
		{
			name: "missing api key",
			cfg: &Config{
				Provider: "openai",
				Model:    "gpt-4o-mini",
				APIKey:   "",
				Lang:     "tr",
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			cfg: &Config{
				Provider: "invalid",
				Model:    "gpt-4o-mini",
				APIKey:   "sk-test",
				Lang:     "tr",
			},
			wantErr: true,
		},
		{
			name: "invalid lang",
			cfg: &Config{
				Provider: "openai",
				Model:    "gpt-4o-mini",
				APIKey:   "sk-test",
				Lang:     "fr",
			},
			wantErr: true,
		},
		{
			name: "empty model",
			cfg: &Config{
				Provider: "openai",
				Model:    "",
				APIKey:   "sk-test",
				Lang:     "tr",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	cfg := &Config{
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
		APIKey:   "sk-test",
		Lang:     "tr",
		Timeout:  30 * time.Second,
	}

	// Save with custom path
	if err := SaveConfigToPath(cfg, configPath); err != nil {
		t.Fatalf("SaveConfigToPath: %v", err)
	}

	// Check file exists and has correct mode
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config file mode = %o, want 0600", info.Mode().Perm())
	}

	// Read and verify content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), `api_key = "sk-test"`) {
		t.Errorf("config missing api_key: %s", string(data))
	}
}

func TestConfigFilePath(t *testing.T) {
	// Test XDG_CONFIG_HOME
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	path, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath: %v", err)
	}
	expected := filepath.Join(tmpDir, "niye", "config.toml")
	if path != expected {
		t.Errorf("ConfigFilePath = %q, want %q", path, expected)
	}

	// Test fallback to ~/.config
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", tmpDir)
	path, err = ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath: %v", err)
	}
	expected = filepath.Join(tmpDir, ".config", "niye", "config.toml")
	if path != expected {
		t.Errorf("ConfigFilePath = %q, want %q", path, expected)
	}
}
