package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Metadata MetadataConfig `mapstructure:"metadata"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
}

// MetadataConfig holds metadata provider configuration.
type MetadataConfig struct {
	TMDB TMDBConfig `mapstructure:"tmdb"`
	TVDB TVDBConfig `mapstructure:"tvdb"`
}

// TMDBConfig holds TMDB API configuration.
type TMDBConfig struct {
	APIKey       string `mapstructure:"api_key"`
	BaseURL      string `mapstructure:"base_url"`
	ImageBaseURL string `mapstructure:"image_base_url"`
	Timeout      int    `mapstructure:"timeout_seconds"`
}

// TVDBConfig holds TVDB API configuration.
type TVDBConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
	Timeout int    `mapstructure:"timeout_seconds"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Path: "./data/slipstream.db",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "console",
		},
		Auth: AuthConfig{
			JWTSecret: "", // Will be generated if empty
		},
		Metadata: MetadataConfig{
			TMDB: TMDBConfig{
				BaseURL:      "https://api.themoviedb.org/3",
				ImageBaseURL: "https://image.tmdb.org/t/p",
				Timeout:      30,
			},
			TVDB: TVDBConfig{
				BaseURL: "https://api4.thetvdb.com/v4",
				Timeout: 30,
			},
		},
	}
}

// Load reads configuration from file and environment variables.
// Priority: environment variables > .env file > config file > defaults
func Load(configPath string) (*Config, error) {
	// Load .env file if it exists (secrets go here)
	// Try multiple locations: current dir, configs dir
	envFiles := []string{".env", "configs/.env"}
	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); err == nil {
			_ = godotenv.Load(envFile) // Ignore error, env vars are optional
			break
		}
	}

	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Config file settings
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("$HOME/.slipstream")
	}

	// Environment variable settings
	v.SetEnvPrefix("SLIPSTREAM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (ignore if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, using defaults + env vars
	}

	// Unmarshal into struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default values in viper
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)

	// Database defaults
	v.SetDefault("database.path", "./data/slipstream.db")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")

	// Auth defaults
	v.SetDefault("auth.jwt_secret", "")

	// Metadata provider defaults
	v.SetDefault("metadata.tmdb.base_url", "https://api.themoviedb.org/3")
	v.SetDefault("metadata.tmdb.image_base_url", "https://image.tmdb.org/t/p")
	v.SetDefault("metadata.tmdb.timeout_seconds", 30)
	v.SetDefault("metadata.tvdb.base_url", "https://api4.thetvdb.com/v4")
	v.SetDefault("metadata.tvdb.timeout_seconds", 30)
}

// Address returns the server address string.
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
