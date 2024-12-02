package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Sync      SyncConfig      `yaml:"sync"`
	Encryption EncryptionConfig `yaml:"encryption"`
	Watch     WatchConfig     `yaml:"watch"`
	Remote    RemoteConfig    `yaml:"remote"`
}

type SyncConfig struct {
	IgnorePatterns []string `yaml:"ignore_patterns"`
	BlockSize      int64    `yaml:"block_size"`
	Compression    bool     `yaml:"compression"`
}

type EncryptionConfig struct {
	Enabled  bool   `yaml:"enabled"`
	KeyFile  string `yaml:"key_file"`
}

type WatchConfig struct {
	DebounceMs int  `yaml:"debounce_ms"`
	Recursive  bool `yaml:"recursive"`
}

type RemoteConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty"`
}

// LoadConfig loads configuration from the specified YAML file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves the configuration to the specified YAML file
func SaveConfig(config *Config, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}
