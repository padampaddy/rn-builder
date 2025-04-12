package main

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
