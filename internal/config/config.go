package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Metadata   MetadataConfig   `mapstructure:"metadata"`
	Indexer    IndexerConfig    `mapstructure:"indexer"`
	AutoSearch AutoSearchConfig `mapstructure:"autosearch"`
	Health     HealthConfig     `mapstructure:"health"`
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
	OMDB OMDBConfig `mapstructure:"omdb"`
}

// TMDBConfig holds TMDB API configuration.
type TMDBConfig struct {
	APIKey                string `mapstructure:"api_key"`
	BaseURL               string `mapstructure:"base_url"`
	ImageBaseURL          string `mapstructure:"image_base_url"`
	Timeout               int    `mapstructure:"timeout_seconds"`
	DisableSearchOrdering bool   `mapstructure:"disable_search_ordering"`
}

// TVDBConfig holds TVDB API configuration.
type TVDBConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
	Timeout int    `mapstructure:"timeout_seconds"`
}

// OMDBConfig holds OMDb API configuration.
type OMDBConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
	Timeout int    `mapstructure:"timeout_seconds"`
}

// IndexerConfig holds indexer-related configuration.
type IndexerConfig struct {
	Cardigann CardigannConfig `mapstructure:"cardigann"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Status    StatusConfig    `mapstructure:"status"`
}

// CardigannConfig holds Cardigann definition system configuration.
type CardigannConfig struct {
	RepositoryURL  string `mapstructure:"repository_url"`  // Default: "https://indexers.prowlarr.com"
	Branch         string `mapstructure:"branch"`          // Default: "master"
	Version        string `mapstructure:"version"`         // Default: "v10"
	DefinitionsDir string `mapstructure:"definitions_dir"` // Default: "./data/definitions"
	CustomDir      string `mapstructure:"custom_dir"`      // Default: "./data/definitions/custom"
	AutoUpdate     bool   `mapstructure:"auto_update"`     // Default: true
	UpdateInterval int    `mapstructure:"update_interval"` // Default: 24 (hours)
	RequestTimeout int    `mapstructure:"request_timeout"` // Default: 60 (seconds)
}

// RateLimitConfig holds rate limiting configuration for indexers.
type RateLimitConfig struct {
	QueryLimit  int `mapstructure:"query_limit"`  // Default: 100
	QueryPeriod int `mapstructure:"query_period"` // Default: 60 (minutes)
	GrabLimit   int `mapstructure:"grab_limit"`   // Default: 25
	GrabPeriod  int `mapstructure:"grab_period"`  // Default: 60 (minutes)
}

// StatusConfig holds indexer health status configuration.
type StatusConfig struct {
	// BackoffMultiplier controls the exponential backoff multiplier.
	BackoffMultiplier float64 `mapstructure:"backoff_multiplier"` // Default: 2.0
	// MaxBackoffHours is the maximum backoff duration in hours.
	MaxBackoffHours int `mapstructure:"max_backoff_hours"` // Default: 3
	// InitialBackoffMinutes is the initial backoff duration in minutes.
	InitialBackoffMinutes int `mapstructure:"initial_backoff_minutes"` // Default: 5
}

// AutoSearchConfig holds automatic search scheduling configuration.
type AutoSearchConfig struct {
	Enabled          bool `mapstructure:"enabled"`           // Default: true
	IntervalHours    int  `mapstructure:"interval_hours"`    // Default: 1 (range: 1-24)
	BackoffThreshold int  `mapstructure:"backoff_threshold"` // Default: 12
	BaseDelayMs      int  `mapstructure:"base_delay_ms"`     // Default: 1000
}

// HealthConfig holds system health monitoring configuration.
type HealthConfig struct {
	DownloadClientCheckInterval time.Duration `mapstructure:"download_client_check_interval"` // Default: 6h
	IndexerCheckInterval        time.Duration `mapstructure:"indexer_check_interval"`         // Default: 6h
	StorageCheckInterval        time.Duration `mapstructure:"storage_check_interval"`         // Default: 1h
	StorageWarningThreshold     float64       `mapstructure:"storage_warning_threshold"`      // Default: 0.20 (20%)
	StorageErrorThreshold       float64       `mapstructure:"storage_error_threshold"`        // Default: 0.05 (5%)
}

// IntervalDuration returns the search interval as a time.Duration.
func (c *AutoSearchConfig) IntervalDuration() time.Duration {
	return time.Duration(c.IntervalHours) * time.Hour
}

// BaseDelayDuration returns the base delay between searches.
func (c *AutoSearchConfig) BaseDelayDuration() time.Duration {
	return time.Duration(c.BaseDelayMs) * time.Millisecond
}

// QueryPeriodDuration returns the query period as a time.Duration.
func (r *RateLimitConfig) QueryPeriodDuration() time.Duration {
	return time.Duration(r.QueryPeriod) * time.Minute
}

// GrabPeriodDuration returns the grab period as a time.Duration.
func (r *RateLimitConfig) GrabPeriodDuration() time.Duration {
	return time.Duration(r.GrabPeriod) * time.Minute
}

// UpdateIntervalDuration returns the update interval as a time.Duration.
func (c *CardigannConfig) UpdateIntervalDuration() time.Duration {
	return time.Duration(c.UpdateInterval) * time.Hour
}

// RequestTimeoutDuration returns the request timeout as a time.Duration.
func (c *CardigannConfig) RequestTimeoutDuration() time.Duration {
	return time.Duration(c.RequestTimeout) * time.Second
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
				BaseURL:               "https://api.themoviedb.org/3",
				ImageBaseURL:          "https://image.tmdb.org/t/p",
				Timeout:               30,
				DisableSearchOrdering: false,
			},
			TVDB: TVDBConfig{
				BaseURL: "https://api4.thetvdb.com/v4",
				Timeout: 30,
			},
			OMDB: OMDBConfig{
				BaseURL: "https://www.omdbapi.com",
				Timeout: 15,
			},
		},
		Indexer: IndexerConfig{
			Cardigann: CardigannConfig{
				RepositoryURL:  "https://indexers.prowlarr.com",
				Branch:         "master",
				Version:        "v10",
				DefinitionsDir: "./data/definitions",
				CustomDir:      "./data/definitions/custom",
				AutoUpdate:     true,
				UpdateInterval: 24,
				RequestTimeout: 60,
			},
			RateLimit: RateLimitConfig{
				QueryLimit:  100,
				QueryPeriod: 60,
				GrabLimit:   25,
				GrabPeriod:  60,
			},
			Status: StatusConfig{
				BackoffMultiplier:     2.0,
				MaxBackoffHours:       3,
				InitialBackoffMinutes: 5,
			},
		},
		AutoSearch: AutoSearchConfig{
			Enabled:          true,
			IntervalHours:    1,
			BackoffThreshold: 12,
			BaseDelayMs:      1000,
		},
		Health: HealthConfig{
			DownloadClientCheckInterval: 6 * time.Hour,
			IndexerCheckInterval:        6 * time.Hour,
			StorageCheckInterval:        1 * time.Hour,
			StorageWarningThreshold:     0.20,
			StorageErrorThreshold:       0.05,
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
	// Note: api_key defaults are empty but must be registered for env var binding to work
	v.SetDefault("metadata.tmdb.api_key", "")
	v.SetDefault("metadata.tmdb.base_url", "https://api.themoviedb.org/3")
	v.SetDefault("metadata.tmdb.image_base_url", "https://image.tmdb.org/t/p")
	v.SetDefault("metadata.tmdb.timeout_seconds", 30)
	v.SetDefault("metadata.tmdb.disable_search_ordering", false)
	v.SetDefault("metadata.tvdb.api_key", "")
	v.SetDefault("metadata.tvdb.base_url", "https://api4.thetvdb.com/v4")
	v.SetDefault("metadata.tvdb.timeout_seconds", 30)
	v.SetDefault("metadata.omdb.api_key", "")
	v.SetDefault("metadata.omdb.base_url", "https://www.omdbapi.com")
	v.SetDefault("metadata.omdb.timeout_seconds", 15)

	// Indexer defaults
	// Cardigann definition system
	v.SetDefault("indexer.cardigann.repository_url", "https://indexers.prowlarr.com")
	v.SetDefault("indexer.cardigann.branch", "master")
	v.SetDefault("indexer.cardigann.version", "v10")
	v.SetDefault("indexer.cardigann.definitions_dir", "./data/definitions")
	v.SetDefault("indexer.cardigann.custom_dir", "./data/definitions/custom")
	v.SetDefault("indexer.cardigann.auto_update", true)
	v.SetDefault("indexer.cardigann.update_interval", 24)
	v.SetDefault("indexer.cardigann.request_timeout", 60)

	// Rate limiting
	v.SetDefault("indexer.rate_limit.query_limit", 100)
	v.SetDefault("indexer.rate_limit.query_period", 60)
	v.SetDefault("indexer.rate_limit.grab_limit", 25)
	v.SetDefault("indexer.rate_limit.grab_period", 60)

	// Status/backoff
	v.SetDefault("indexer.status.backoff_multiplier", 2.0)
	v.SetDefault("indexer.status.max_backoff_hours", 3)
	v.SetDefault("indexer.status.initial_backoff_minutes", 5)

	// AutoSearch defaults
	v.SetDefault("autosearch.enabled", true)
	v.SetDefault("autosearch.interval_hours", 1)
	v.SetDefault("autosearch.backoff_threshold", 12)
	v.SetDefault("autosearch.base_delay_ms", 1000)

	// Health check defaults
	v.SetDefault("health.download_client_check_interval", 6*time.Hour)
	v.SetDefault("health.indexer_check_interval", 6*time.Hour)
	v.SetDefault("health.storage_check_interval", 1*time.Hour)
	v.SetDefault("health.storage_warning_threshold", 0.20)
	v.SetDefault("health.storage_error_threshold", 0.05)
}

// Address returns the server address string.
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ToManagerConfig converts IndexerConfig to cardigann.ManagerConfig compatible values.
// Returns RepositoryConfig values, CacheConfig values, and manager settings.
func (ic *IndexerConfig) ToManagerConfigValues() (
	repoURL, branch, version, userAgent string,
	requestTimeout time.Duration,
	definitionsDir, customDir string,
	autoUpdate bool,
	updateInterval time.Duration,
) {
	c := ic.Cardigann
	repoURL = c.RepositoryURL
	branch = c.Branch
	version = c.Version
	userAgent = "SlipStream/1.0"
	requestTimeout = c.RequestTimeoutDuration()
	definitionsDir = c.DefinitionsDir
	customDir = c.CustomDir
	autoUpdate = c.AutoUpdate
	updateInterval = c.UpdateIntervalDuration()
	return
}
