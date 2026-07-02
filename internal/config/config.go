// This file defines the configuration structure for the application.
package config

import (
	// use Viper for loading the config.yml file.
	"strings"

	"github.com/spf13/viper"
)

// MetadataConfig holds configuration for metadata operations.
type MetadataConfig struct {
	CoverMode        string        `mapstructure:"cover_mode" yaml:"cover_mode"`
	CoverFailureMode string        `mapstructure:"cover_failure_mode" yaml:"cover_failure_mode"`
	AniList          AniListConfig `mapstructure:"anilist" yaml:"anilist"`
}

// AniListConfig holds AniList-specific metadata configuration.
type AniListConfig struct {
	ExcludeSpoilerTags bool `mapstructure:"exclude_spoiler_tags" yaml:"exclude_spoiler_tags"`
}

// Config holds all configuration settings for the application.
// It maps directly to the structure of config.yml.
type Config struct {
	Port         int `mapstructure:"port"`
	ScanInterval int `mapstructure:"scan_interval"`
	Database     struct {
		Path string `mapstructure:"path"`
	} `mapstructure:"database"`
	Library struct {
		Path string `mapstructure:"path"`
	} `mapstructure:"library"`
	Plugins struct {
		Path          string `mapstructure:"path"`
		UnloadTimeout int    `mapstructure:"unload_timeout"` // Minutes of inactivity before unloading
	} `mapstructure:"plugins"`
	Metadata MetadataConfig `mapstructure:"metadata"`
}

// Load reads configuration from a file named "config.yml" in the
// current directory and unmarshals it into a Config struct.
func Load() (*Config, error) {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yml")    // or "yaml"
	viper.AddConfigPath(".")      // looking for config in the current directory

	// --- Environment Variable Overrides ---
	// This tells Viper to look for environment variables with a "MANGO_" prefix.
	// e.g., MANGO_DATABASE_PATH will override the `database.path` key.
	viper.SetEnvPrefix("MANGO")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("port", 8080)
	viper.SetDefault("scan_interval", 0)
	viper.SetDefault("database.path", "./mango.db")
	viper.SetDefault("library.path", "./manga")
	viper.SetDefault("plugins.path", "../mango-go-plugins")
	viper.SetDefault("plugins.unload_timeout", 30)
	viper.SetDefault("metadata.cover_mode", "direct")
	viper.SetDefault("metadata.cover_failure_mode", "ignore")
	viper.SetDefault("metadata.anilist.exclude_spoiler_tags", true)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error and use defaults
		} else {
			// Config file was found but another error was produced
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
