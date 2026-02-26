package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application
type Config struct {
	API        APIConfig        `mapstructure:"api" yaml:"api"`
	Player     PlayerConfig     `mapstructure:"player" yaml:"player"`
	Providers  ProvidersConfig  `mapstructure:"providers" yaml:"providers"`
	Tracker    TrackerConfig    `mapstructure:"tracker" yaml:"tracker"`
	Downloads  DownloadsConfig  `mapstructure:"downloads" yaml:"downloads"`
	UI         UIConfig         `mapstructure:"ui" yaml:"ui"`
	WatchParty WatchPartyConfig `mapstructure:"watchparty" yaml:"watchparty"`
	Cache      CacheConfig      `mapstructure:"cache" yaml:"cache"`
	Database   DatabaseConfig   `mapstructure:"database" yaml:"database"`
	Logging    LoggingConfig    `mapstructure:"logging" yaml:"logging"`
	Network    NetworkConfig    `mapstructure:"network" yaml:"network"`
	Advanced   AdvancedConfig   `mapstructure:"advanced" yaml:"advanced"`

	// Internal fields
	configPath string
}

// APIConfig contains API server settings
type APIConfig struct {
	BaseURL string        `mapstructure:"base_url"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// PlayerConfig contains video player settings
type PlayerConfig struct {
	Binary          string        `mapstructure:"binary"`
	MPVArgs         []string      `mapstructure:"mpv_args"`
	Quality         string        `mapstructure:"quality"`
	Resume          bool          `mapstructure:"resume"`
	SubtitleLang    string        `mapstructure:"subtitle_language"`
	AutoSubtitles   bool          `mapstructure:"auto_subtitles"`
	AudioPreference string        `mapstructure:"audio_preference"`
	LoadUserConfig  bool          `mapstructure:"load_user_config"`
	IPCTimeout      time.Duration `mapstructure:"ipc_timeout"`
}

// ProvidersConfig contains provider settings
type ProvidersConfig struct {
	Default             DefaultProviders  `mapstructure:"default" yaml:"default"`
	Priority            PriorityProviders `mapstructure:"priority" yaml:"priority"`
	HealthCheckInterval time.Duration     `mapstructure:"health_check_interval" yaml:"health_check_interval"`
	AutoFailover        bool              `mapstructure:"auto_failover" yaml:"auto_failover"`
	AllAnime            ProviderSettings  `mapstructure:"allanime" yaml:"allanime"`
	HiAnime             ProviderSettings  `mapstructure:"hianime" yaml:"hianime"`
	SFlix               ProviderSettings  `mapstructure:"sflix" yaml:"sflix"`
	FlixHQ              ProviderSettings  `mapstructure:"flixhq" yaml:"flixhq"`
	HDRezka             ProviderSettings  `mapstructure:"hdrezka" yaml:"hdrezka"`
	Comix               ProviderSettings  `mapstructure:"comix" yaml:"comix"`
	Lua                 ProviderSettings  `mapstructure:"lua" yaml:"lua"`
}

// DefaultProviders specifies default provider for each media type
type DefaultProviders struct {
	Anime       string `mapstructure:"anime" yaml:"anime"`
	MoviesAndTV string `mapstructure:"movies_and_tv" yaml:"movies_and_tv"` // Combined field for movies and TV
}

// PriorityProviders specifies provider priority order
type PriorityProviders struct {
	Anime  []string `mapstructure:"anime"`
	Movies []string `mapstructure:"movies"`
	TV     []string `mapstructure:"tv"`
}

// ProviderSettings contains provider-specific settings
type ProviderSettings struct {
	Mode       string        `mapstructure:"mode"`       // "local" or "remote" (Default: "local")
	RemoteURL  string        `mapstructure:"remote_url"` // Target API URL if mode is remote
	Enabled    bool          `mapstructure:"enabled"`
	BaseURL    string        `mapstructure:"base_url"`
	APIURL     string        `mapstructure:"api_url"`
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
	RateLimit  int           `mapstructure:"rate_limit"`
}

// TrackerConfig contains tracker settings
type TrackerConfig struct {
	AniList AniListConfig `mapstructure:"anilist"`
}

// AniListConfig contains AniList-specific settings
type AniListConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	AutoSync      bool          `mapstructure:"auto_sync"`
	SyncThreshold float64       `mapstructure:"sync_threshold"`
	AutoComplete  bool          `mapstructure:"auto_complete"`
	SyncInterval  time.Duration `mapstructure:"sync_interval"`
	RedirectURI   string        `mapstructure:"redirect_uri"`
	ServerPort    int           `mapstructure:"server_port"`
}

// DownloadsConfig contains download settings
type DownloadsConfig struct {
	Path                  string   `mapstructure:"path"`
	Concurrent            int      `mapstructure:"concurrent"`
	ConcurrentSegments    int      `mapstructure:"concurrent_segments"`
	EmbedSubtitles        bool     `mapstructure:"embed_subtitles"`
	SubtitleLanguages     []string `mapstructure:"subtitle_languages"`
	AutoResume            bool     `mapstructure:"auto_resume"`
	KeepPartial           bool     `mapstructure:"keep_partial"`
	FilenameTemplate      string   `mapstructure:"filename_template"`
	AnimeFilenameTemplate string   `mapstructure:"anime_filename_template"`
	MovieFilenameTemplate string   `mapstructure:"movie_filename_template"`
	MaxSpeed              int64    `mapstructure:"max_speed"`
	MinFreeSpace          int      `mapstructure:"min_free_space"`
}

// UIConfig contains UI settings
type UIConfig struct {
	Theme            string            `mapstructure:"theme"`
	PreviewImages    bool              `mapstructure:"preview_images"`
	PreviewMethod    string            `mapstructure:"preview_method"`
	MangaMethod      string            `mapstructure:"manga_method"`
	PreviewSize      PreviewSize       `mapstructure:"preview_size"`
	ShowProgress     bool              `mapstructure:"show_progress"`
	Compact          bool              `mapstructure:"compact"`
	Keybindings      map[string]string `mapstructure:"keybindings"`
	DateFormat       string            `mapstructure:"date_format"`
	TimeFormat       string            `mapstructure:"time_format"`
	FuzzyFinder      string            `mapstructure:"fuzzy_finder"`
	ShowLoading      bool              `mapstructure:"show_loading"`
	DefaultMediaType string            `mapstructure:"default_media_type"` // movie_tv, anime, or manga
}

// PreviewSize contains preview image dimensions
type PreviewSize struct {
	Width  int `mapstructure:"width"`
	Height int `mapstructure:"height"`
}

// WatchPartyConfig contains WatchParty settings
type WatchPartyConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	DefaultProxy    string `mapstructure:"default_proxy"`
	AutoOpenBrowser bool   `mapstructure:"auto_open_browser"`
	DefaultOrigin   string `mapstructure:"default_origin"`
}

// CacheConfig contains cache settings
type CacheConfig struct {
	Enabled       bool     `mapstructure:"enabled"`
	Path          string   `mapstructure:"path"`
	TTL           CacheTTL `mapstructure:"ttl"`
	MaxSize       int      `mapstructure:"max_size"`
	CleanupOnExit bool     `mapstructure:"cleanup_on_exit"`
}

// CacheTTL contains TTL for different cache types
type CacheTTL struct {
	Metadata      time.Duration `mapstructure:"metadata"`
	Images        time.Duration `mapstructure:"images"`
	SearchResults time.Duration `mapstructure:"search_results"`
	StreamURLs    time.Duration `mapstructure:"stream_urls"`
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	Path           string `mapstructure:"path"`
	WALMode        bool   `mapstructure:"wal_mode"`
	MaxConnections int    `mapstructure:"max_connections"`
	AutoVacuum     bool   `mapstructure:"auto_vacuum"`
	BackupOnExit   bool   `mapstructure:"backup_on_exit"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
	Format     string `mapstructure:"format"`
	Color      bool   `mapstructure:"color"`
}

// NetworkConfig contains network settings
type NetworkConfig struct {
	Timeout         time.Duration `mapstructure:"timeout"`
	HTTP2           bool          `mapstructure:"http2"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	IdleConnTimeout time.Duration `mapstructure:"idle_conn_timeout"`
	UserAgent       string        `mapstructure:"user_agent"`
	Proxy           string        `mapstructure:"proxy"`
	VerifyTLS       bool          `mapstructure:"verify_tls"`
	DNSServers      []string      `mapstructure:"dns_servers"`
}

// AdvancedConfig contains advanced settings
type AdvancedConfig struct {
	Experimental  bool            `mapstructure:"experimental"`
	Debug         bool            `mapstructure:"debug"`
	NoTelemetry   bool            `mapstructure:"no_telemetry"`
	Profiling     []string        `mapstructure:"profiling"`
	MaxGoroutines int             `mapstructure:"max_goroutines"`
	Clipboard     ClipboardConfig `mapstructure:"clipboard"`
}

// ClipboardConfig contains clipboard settings
type ClipboardConfig struct {
	Command string `mapstructure:"command"`
}

// Load loads configuration from file, environment, and defaults
func Load(configPath string) (*Config, *viper.Viper, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Add default config paths
		v.AddConfigPath(getConfigDir())
		v.AddConfigPath(".")
	}

	// Enable environment variable support
	v.SetEnvPrefix("GREG")
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found; using defaults
	}

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Store config path
	cfg.configPath = v.ConfigFileUsed()

	// Expand paths
	cfg.Downloads.Path = expandPath(cfg.Downloads.Path)
	cfg.Cache.Path = expandPath(cfg.Cache.Path)
	cfg.Database.Path = expandPath(cfg.Database.Path)
	cfg.Logging.File = expandPath(cfg.Logging.File)

	return &cfg, v, nil
}

// Save saves the configuration to the file
func (c *Config) Save() error {
	// Determine config path
	configPath := c.configPath
	if configPath == "" {
		configPath = filepath.Join(getConfigDir(), "config.yaml")
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	// Debug: Print what we're about to save
	fmt.Printf("DEBUG: Saving config to: %s\n", configPath)
	fmt.Printf("DEBUG: providers.default.anime: %s\n", c.Providers.Default.Anime)
	fmt.Printf("DEBUG: providers.default.movies_and_tv: %s\n", c.Providers.Default.MoviesAndTV)

	// Marshal the config to YAML bytes directly
	yamlData, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, yamlData, 0o644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", configPath, err)
	}

	fmt.Printf("DEBUG: Config saved successfully to: %s\n", configPath)
	return nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// API defaults
	v.SetDefault("api.base_url", "http://localhost:8080")
	v.SetDefault("api.timeout", 30*time.Second)

	// Player defaults
	v.SetDefault("player.binary", "mpv")
	v.SetDefault("player.quality", "1080p")
	v.SetDefault("player.resume", true)
	v.SetDefault("player.subtitle_language", "en")
	v.SetDefault("player.auto_subtitles", true)
	v.SetDefault("player.audio_preference", "sub")
	v.SetDefault("player.load_user_config", true)
	v.SetDefault("player.ipc_timeout", 5*time.Second)

	// Provider defaults
	v.SetDefault("providers.default.anime", "hianime")
	v.SetDefault("providers.default.movies_and_tv", "sflix") // Combined default for movies and TV
	v.SetDefault("providers.health_check_interval", 5*time.Minute)
	v.SetDefault("providers.auto_failover", true)

	// AllAnime defaults (API-based)
	v.SetDefault("providers.allanime.enabled", true)
	v.SetDefault("providers.allanime.mode", "local")

	// HiAnime defaults (API-based)
	v.SetDefault("providers.hianime.enabled", true)
	v.SetDefault("providers.hianime.mode", "local")

	// SFlix defaults (API-based)
	v.SetDefault("providers.sflix.enabled", true)
	v.SetDefault("providers.sflix.mode", "local")

	// FlixHQ defaults (API-based)
	v.SetDefault("providers.flixhq.enabled", true)
	v.SetDefault("providers.flixhq.mode", "local")

	// HDRezka defaults (API-based)
	v.SetDefault("providers.hdrezka.enabled", true)
	v.SetDefault("providers.hdrezka.mode", "local")

	// Comix defaults (API-based)
	v.SetDefault("providers.comix.enabled", true)
	v.SetDefault("providers.comix.mode", "local")

	// Lua defaults (Local)
	v.SetDefault("providers.lua.enabled", true)
	v.SetDefault("providers.lua.mode", "local")

	// Tracker defaults
	v.SetDefault("tracker.anilist.enabled", true)
	v.SetDefault("tracker.anilist.auto_sync", true)
	v.SetDefault("tracker.anilist.sync_threshold", 0.85)
	v.SetDefault("tracker.anilist.auto_complete", true)
	v.SetDefault("tracker.anilist.sync_interval", 5*time.Minute)
	v.SetDefault("tracker.anilist.redirect_uri", "http://localhost:8000/oauth/callback")
	v.SetDefault("tracker.anilist.server_port", 8000)

	// Download defaults
	v.SetDefault("downloads.path", filepath.Join(getVideosDir(), "greg"))
	v.SetDefault("downloads.concurrent", 3)
	v.SetDefault("downloads.concurrent_segments", 5)
	v.SetDefault("downloads.embed_subtitles", true)
	v.SetDefault("downloads.subtitle_languages", []string{"en"})
	v.SetDefault("downloads.auto_resume", true)
	v.SetDefault("downloads.keep_partial", true)
	v.SetDefault("downloads.filename_template", "{title} - S{season:02d}E{episode:02d} [{quality}]")
	v.SetDefault("downloads.anime_filename_template", "{title} - {episode:03d} [{quality}]")
	v.SetDefault("downloads.movie_filename_template", "{title} ({year}) [{quality}]")
	v.SetDefault("downloads.max_speed", 0)
	v.SetDefault("downloads.min_free_space", 5)

	// UI defaults
	v.SetDefault("ui.theme", "default")
	v.SetDefault("ui.preview_images", true)
	v.SetDefault("ui.preview_method", "auto")
	v.SetDefault("ui.manga_method", "sixel")
	v.SetDefault("ui.preview_size.width", 40)
	v.SetDefault("ui.preview_size.height", 20)
	v.SetDefault("ui.show_progress", true)
	v.SetDefault("ui.compact", false)
	v.SetDefault("ui.date_format", "2006-01-02")
	v.SetDefault("ui.time_format", "15:04")
	v.SetDefault("ui.fuzzy_finder", "builtin")
	v.SetDefault("ui.show_loading", false)

	// WatchParty defaults
	v.SetDefault("watchparty.enabled", true)
	v.SetDefault("watchparty.default_proxy", "")
	v.SetDefault("watchparty.auto_open_browser", true)
	v.SetDefault("watchparty.default_origin", "https://videostr.net")

	// Cache defaults
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.path", filepath.Join(getCacheDir(), "greg"))
	v.SetDefault("cache.ttl.metadata", 5*time.Minute)
	v.SetDefault("cache.ttl.images", 24*time.Hour)
	v.SetDefault("cache.ttl.search_results", 10*time.Minute)
	v.SetDefault("cache.ttl.stream_urls", 1*time.Minute)
	v.SetDefault("cache.max_size", 500)
	v.SetDefault("cache.cleanup_on_exit", false)

	// Database defaults
	v.SetDefault("database.path", filepath.Join(getDataDir(), "greg", "greg.db"))
	v.SetDefault("database.wal_mode", true)
	v.SetDefault("database.max_connections", 10)
	v.SetDefault("database.auto_vacuum", true)
	v.SetDefault("database.backup_on_exit", false)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.file", filepath.Join(getStateDir(), "greg", "greg.log"))
	v.SetDefault("logging.max_size", 10)
	v.SetDefault("logging.max_backups", 3)
	v.SetDefault("logging.max_age", 7)
	v.SetDefault("logging.compress", true)
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.color", true)

	// Network defaults
	v.SetDefault("network.timeout", 30*time.Second)
	v.SetDefault("network.http2", true)
	v.SetDefault("network.max_idle_conns", 100)
	v.SetDefault("network.idle_conn_timeout", 90*time.Second)
	v.SetDefault("network.user_agent", "greg/1.0.0")
	v.SetDefault("network.verify_tls", true)

	// Advanced defaults
	v.SetDefault("advanced.experimental", false)
	v.SetDefault("advanced.debug", false)
	v.SetDefault("advanced.no_telemetry", true)
	v.SetDefault("advanced.max_goroutines", 100)
	v.SetDefault("advanced.clipboard.command", "") // Empty by default, user can set to "clip.exe" for WSL
}

// getConfigDir returns the config directory path
func getConfigDir() string {
	// GREG_* override for testing
	if gregConfig := os.Getenv("GREG_CONFIG_HOME"); gregConfig != "" {
		return gregConfig
	}
	// Windows: Use APPDATA
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		return filepath.Join(appdata, "greg")
	}
	// Unix: Use XDG_CONFIG_HOME or fallback to ~/.config
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "greg")
	}
	return filepath.Join(getHomeDir(), ".config", "greg")
}

// GetConfigDir returns the config directory path (exported version)
func GetConfigDir() string {
	return getConfigDir()
}

// getCacheDir returns the cache directory path
func getCacheDir() string {
	// Windows: Use LOCALAPPDATA for cache
	if localappdata := os.Getenv("LOCALAPPDATA"); localappdata != "" {
		return filepath.Join(localappdata, "greg")
	}
	// Unix: Use XDG_CACHE_HOME or fallback to ~/.cache
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return xdg
	}
	return filepath.Join(getHomeDir(), ".cache")
}

// getDataDir returns the data directory path
func getDataDir() string {
	// Windows: Use APPDATA for persistent data
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		return filepath.Join(appdata, "greg")
	}
	// Unix: Use XDG_DATA_HOME or fallback to ~/.local/share
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return xdg
	}
	return filepath.Join(getHomeDir(), ".local", "share")
}

// getStateDir returns the state directory path
func getStateDir() string {
	// GREG_* override for testing
	if gregState := os.Getenv("GREG_STATE_HOME"); gregState != "" {
		return gregState
	}
	// Windows: Use LOCALAPPDATA for state/logs
	if localappdata := os.Getenv("LOCALAPPDATA"); localappdata != "" {
		return filepath.Join(localappdata, "greg")
	}
	// Unix: Use XDG_STATE_HOME or fallback to ~/.local/state
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return xdg
	}
	return filepath.Join(getHomeDir(), ".local", "state")
}

// getVideosDir returns the videos directory path
func getVideosDir() string {
	// GREG_* override for testing
	if gregVideos := os.Getenv("GREG_VIDEOS_DIR"); gregVideos != "" {
		return gregVideos
	}
	// Windows: Use USERPROFILE\Videos
	if userprofile := os.Getenv("USERPROFILE"); userprofile != "" {
		return filepath.Join(userprofile, "Videos")
	}
	// macOS detection (use Movies not Videos)
	if runtime.GOOS == "darwin" {
		if xdg := os.Getenv("XDG_VIDEOS_DIR"); xdg != "" {
			return xdg
		}
		return filepath.Join(getHomeDir(), "Movies")
	}
	// Linux: XDG_VIDEOS_DIR or ~/Videos
	if xdg := os.Getenv("XDG_VIDEOS_DIR"); xdg != "" {
		return xdg
	}
	return filepath.Join(getHomeDir(), "Videos")
}

// getHomeDir returns the user's home directory
func getHomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		return home
	}
	return "."
}

// InitializeDirs creates all required greg directories if they don't exist
func InitializeDirs() error {
	dirs := []string{
		getConfigDir(),                        // CONFIG_HOME/greg/ (getConfigDir already includes /greg)
		filepath.Join(getStateDir(), "greg"),  // STATE_HOME/greg/ (for logging.file)
		filepath.Join(getDataDir(), "greg"),   // DATA_HOME/greg/ (for database.path)
		filepath.Join(getCacheDir(), "greg"),  // CACHE_HOME/greg/ (for cache.path)
		filepath.Join(getVideosDir(), "greg"), // VIDEOS_DIR/greg/ (for downloads.path)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if len(path) == 0 {
		return path
	}
	if path[0] == '~' {
		return filepath.Join(getHomeDir(), path[1:])
	}
	return path
}

// SetDefaults sets default configuration values
// This function is exported for use in other packages
func SetDefaults(v *viper.Viper) {
	setDefaults(v)
}

// SaveDefaultConfig saves the default configuration to a file
func SaveDefaultConfig(configPath string) error {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config name and type
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Write the config to file
	return v.WriteConfigAs(configPath)
}
