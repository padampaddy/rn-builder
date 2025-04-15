package main

import (
	"fmt"
	"os"
	"path/filepath"
	"gopkg.in/yaml.v3"
)

type Config struct {
	RootPath          string `yaml:"root_path"`
	BuildVersion      string `yaml:"build_version"`
	Platform          string `yaml:"platform"`
	DriveFolderID     string `yaml:"drive_folder_id"`
	SkipUpload        bool   `yaml:"skip_upload"`
	SkipDeps          bool   `yaml:"skip_deps"`
	AppleID           string `yaml:"apple_id"`           // For TestFlight upload
	TeamID            string `yaml:"team_id"`            // For TestFlight upload (non-main/provider)
	ReleaseChannel    string `yaml:"release_channel"`    // Keep if used by expo prebuild or other logic
	GoogleCredentials string `yaml:"google_credentials"` // Path to credentials file
	Android           struct {
		BuildType string `yaml:"build_type"` // e.g., "Release", "Debug", or flavor like "ProductionRelease"
		// Add flavor if needed: Flavor string `yaml:"flavor"`
	} `yaml:"android"`
	IOS struct {
		Enterprise  bool   `yaml:"enterprise"`   // Use Enterprise distribution?
		Scheme      string `yaml:"scheme"`       // Optional: Override auto-detected scheme
		ProjectName string `yaml:"project_name"` // Optional: Override auto-detected workspace/project name
	} `yaml:"ios"`
}

// SaveConfig saves the configuration to a YAML file
func (c *Config) SaveConfig(filename string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}
