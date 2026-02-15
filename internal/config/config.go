package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the CLI configuration.
type Config struct {
	CurrentProfile string            `yaml:"current-profile" mapstructure:"current-profile"`
	Profiles       map[string]Profile `yaml:"profiles" mapstructure:"profiles"`
	Output         OutputConfig      `yaml:"output" mapstructure:"output"`
}

// Profile represents a named configuration profile.
type Profile struct {
	APIURL string `yaml:"api-url" mapstructure:"api-url"`
	Token  string `yaml:"token" mapstructure:"token"`
}

// OutputConfig represents output formatting configuration.
type OutputConfig struct {
	Format string `yaml:"format" mapstructure:"format"`
}

// Default values.
const (
	DefaultConfigFile = "~/.s0/config.yaml"
	DefaultProfile    = "default"
	DefaultAPIURL     = "https://api.sandbox0.ai"
	DefaultFormat     = "table"
)

// Environment variables (same as sdk-go/sdk-js/sdk-py).
const (
	EnvToken   = "SANDBOX0_TOKEN"
	EnvBaseURL = "SANDBOX0_BASE_URL"
)

var (
	cfgFile string
	profile string
	apiURL  string
	token   string
)

// GetConfigFile returns a pointer to the config file path variable for flag binding.
func GetConfigFile() *string {
	return &cfgFile
}

// SetConfigFile sets the config file path.
func SetConfigFile(path string) {
	cfgFile = path
}

// GetProfileVar returns a pointer to the profile variable for flag binding.
func GetProfileVar() *string {
	return &profile
}

// GetAPIURLVar returns a pointer to the apiURL variable for flag binding.
func GetAPIURLVar() *string {
	return &apiURL
}

// GetTokenVar returns a pointer to the token variable for flag binding.
func GetTokenVar() *string {
	return &token
}

// SetProfile sets the active profile.
func SetProfile(p string) {
	profile = p
}

// SetAPIURL overrides the API URL.
func SetAPIURL(url string) {
	apiURL = url
}

// SetToken overrides the token.
func SetToken(t string) {
	token = t
}

// Load loads the configuration from file.
func Load() (*Config, error) {
	configPath := expandPath(cfgFile)
	if configPath == "" {
		configPath = expandPath(DefaultConfigFile)
	}

	v := viper.New()
	v.SetConfigFile(configPath)

	// Set defaults
	v.SetDefault("current-profile", DefaultProfile)
	v.SetDefault("output.format", DefaultFormat)

	// Read config if exists
	if _, err := os.Stat(configPath); err == nil {
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Initialize profiles map if empty
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}

	return &cfg, nil
}

// GetActiveProfile returns the active profile name.
func (c *Config) GetActiveProfile() string {
	if profile != "" {
		return profile
	}
	if c.CurrentProfile != "" {
		return c.CurrentProfile
	}
	return DefaultProfile
}

// GetProfile returns the specified profile configuration.
func (c *Config) GetProfile(name string) (*Profile, error) {
	p, ok := c.Profiles[name]
	if !ok {
		// Return default profile with defaults
		return &Profile{
			APIURL: DefaultAPIURL,
		}, nil
	}
	return &p, nil
}

// GetAPIURL returns the API URL with override and env support.
// Priority: --api-url flag > SANDBOX0_BASE_URL env > profile config > default
func (p *Profile) GetAPIURL() string {
	// Command line flag takes highest priority
	if apiURL != "" {
		return apiURL
	}
	// Check standard environment variable
	if envURL := os.Getenv("SANDBOX0_BASE_URL"); envURL != "" {
		return envURL
	}
	// Profile config
	if p.APIURL != "" {
		return p.APIURL
	}
	return DefaultAPIURL
}

// GetToken returns the token with override and env support.
// Priority: --token flag > SANDBOX0_TOKEN env > profile config (with env expansion)
func (p *Profile) GetToken() string {
	// Command line flag takes highest priority
	if token != "" {
		return token
	}
	// Check standard environment variable
	if envToken := os.Getenv("SANDBOX0_TOKEN"); envToken != "" {
		return envToken
	}
	// Profile config with env var expansion support
	return expandEnvVars(p.Token)
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// expandEnvVars expands environment variables in the format ${VAR} or $VAR.
func expandEnvVars(s string) string {
	if s == "" {
		return ""
	}

	// Handle ${VAR} format
	for {
		start := strings.Index(s, "${")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "}")
		if end == -1 {
			break
		}
		end += start

		envVar := s[start+2 : end]
		envValue := os.Getenv(envVar)
		s = s[:start] + envValue + s[end+1:]
	}

	return s
}
