package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var guiApp fyne.App

// updateUIFromConfig updates all UI elements based on the provided config
func updateUIFromConfig(config *Config, entries map[string]interface{}) {
	if e, ok := entries["rootPath"].(*widget.Entry); ok {
		e.SetText(config.RootPath)
	}
	if e, ok := entries["version"].(*widget.Entry); ok {
		e.SetText(config.BuildVersion)
	}
	if r, ok := entries["platform"].(*widget.RadioGroup); ok {
		r.SetSelected(config.Platform)
	}
	if c, ok := entries["skipUpload"].(*widget.Check); ok {
		c.SetChecked(config.SkipUpload)
	}
	if c, ok := entries["skipDeps"].(*widget.Check); ok {
		c.SetChecked(config.SkipDeps)
	}
	if e, ok := entries["androidBuildType"].(*widget.Entry); ok {
		e.SetText(config.Android.BuildType)
	}
	if e, ok := entries["driveFolder"].(*widget.Entry); ok {
		e.SetText(config.DriveFolderID)
	}
	if e, ok := entries["googleCreds"].(*widget.Entry); ok {
		e.SetText(config.GoogleCredentials)
	}
	if c, ok := entries["iosEnterprise"].(*widget.Check); ok {
		c.SetChecked(config.IOS.Enterprise)
	}
	if e, ok := entries["iosScheme"].(*widget.Entry); ok {
		e.SetText(config.IOS.Scheme)
	}
	if e, ok := entries["iosProjectName"].(*widget.Entry); ok {
		e.SetText(config.IOS.ProjectName)
	}
	if e, ok := entries["appleID"].(*widget.Entry); ok {
		e.SetText(config.AppleID)
	}
	if e, ok := entries["teamID"].(*widget.Entry); ok {
		e.SetText(config.TeamID)
	}
}

// getConfigFromUI creates a Config struct from the current UI state
func getConfigFromUI(entries map[string]interface{}) Config {
	config := Config{}

	if e, ok := entries["rootPath"].(*widget.Entry); ok {
		config.RootPath = e.Text
	}
	if e, ok := entries["version"].(*widget.Entry); ok {
		config.BuildVersion = e.Text
	}
	if r, ok := entries["platform"].(*widget.RadioGroup); ok {
		config.Platform = r.Selected
	}
	if c, ok := entries["skipUpload"].(*widget.Check); ok {
		config.SkipUpload = c.Checked
	}
	if c, ok := entries["skipDeps"].(*widget.Check); ok {
		config.SkipDeps = c.Checked
	}
	if e, ok := entries["androidBuildType"].(*widget.Entry); ok {
		config.Android.BuildType = e.Text
	}
	if e, ok := entries["driveFolder"].(*widget.Entry); ok {
		config.DriveFolderID = e.Text
	}
	if e, ok := entries["googleCreds"].(*widget.Entry); ok {
		config.GoogleCredentials = e.Text
	}
	if c, ok := entries["iosEnterprise"].(*widget.Check); ok {
		config.IOS.Enterprise = c.Checked
	}
	if e, ok := entries["iosScheme"].(*widget.Entry); ok {
		config.IOS.Scheme = e.Text
	}
	if e, ok := entries["iosProjectName"].(*widget.Entry); ok {
		config.IOS.ProjectName = e.Text
	}
	if e, ok := entries["appleID"].(*widget.Entry); ok {
		config.AppleID = e.Text
	}
	if e, ok := entries["teamID"].(*widget.Entry); ok {
		config.TeamID = e.Text
	}

	return config
}

func main() {
	guiApp = app.NewWithID("com.mps.rn_builder")
	window := guiApp.NewWindow("React Native Builder")
	window.Resize(fyne.NewSize(800, 800)) // Set a reasonable initial size

	// Create a map to store UI elements for easy access
	uiEntries := make(map[string]interface{})

	// --- Create UI Widgets ---
	// Root Path
	rootPathEntry := widget.NewEntry()
	uiEntries["rootPath"] = rootPathEntry
	rootPathButton := widget.NewButton("Browse...", func() {
		dialog.NewFolderOpen(func(reader fyne.ListableURI, err error) {
			if err == nil && reader != nil {
				rootPathEntry.SetText(reader.Path())
			}
		}, window).Show()
	})

	// Build Version
	versionEntry := widget.NewEntry()
	uiEntries["version"] = versionEntry
	versionEntry.Validator = func(s string) error {
		if !isValidVersion(s) { // Reuse your validation function
			return fmt.Errorf("invalid format (e.g., 1.2.3)")
		}
		return nil
	}

	// Platform
	platformRadio := widget.NewRadioGroup([]string{"All", "Android", "iOS"}, nil)
	uiEntries["platform"] = platformRadio
	platformRadio.SetSelected("All") // Default selection

	// Options
	skipUploadCheck := widget.NewCheck("Skip Uploads", nil)
	uiEntries["skipUpload"] = skipUploadCheck
	skipDepsCheck := widget.NewCheck("Skip Dependencies", nil)
	uiEntries["skipDeps"] = skipDepsCheck

	// Android Specific
	androidBuildTypeEntry := widget.NewEntry()
	uiEntries["androidBuildType"] = androidBuildTypeEntry
	androidBuildTypeEntry.SetText("Release") // Default
	driveFolderEntry := widget.NewEntry()
	uiEntries["driveFolder"] = driveFolderEntry
	googleCredsEntry := widget.NewEntry()
	uiEntries["googleCreds"] = googleCredsEntry
	googleCredsButton := widget.NewButton("Browse...", func() {
		dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				googleCredsEntry.SetText(reader.URI().Path())
				reader.Close()
			}
		}, window).Show()
	})

	// iOS Specific
	iosEnterpriseCheck := widget.NewCheck("Enterprise Build (iOS)", nil)
	uiEntries["iosEnterprise"] = iosEnterpriseCheck
	iosSchemeEntry := widget.NewEntry()
	uiEntries["iosScheme"] = iosSchemeEntry
	iosSchemeEntry.PlaceHolder = "Optional: Auto-detected"
	iosProjectNameEntry := widget.NewEntry()
	uiEntries["iosProjectName"] = iosProjectNameEntry
	iosProjectNameEntry.PlaceHolder = "Optional: Auto-detected"
	appleIDEntry := widget.NewEntry()
	uiEntries["appleID"] = appleIDEntry
	appleIDEntry.PlaceHolder = "Apple ID (email)"
	teamIDEntry := widget.NewEntry()
	uiEntries["teamID"] = teamIDEntry
	teamIDEntry.PlaceHolder = "Apple Team ID (Optional)"

	// Config buttons
	saveConfigButton := widget.NewButton("Save Config", func() {
		config := getConfigFromUI(uiEntries)
		configPath := filepath.Join(".", defaultConfig)
		if err := config.SaveConfig(configPath); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save config: %w", err), window)
			return
		}
		dialog.ShowInformation("Success", "Configuration saved successfully", window)
	})

	loadConfigButton := widget.NewButton("Load Config", func() {
		configPath := filepath.Join(".", defaultConfig)
		config, err := LoadConfig(configPath)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to load config: %w", err), window)
			return
		}
		updateUIFromConfig(config, uiEntries)
		dialog.ShowInformation("Success", "Configuration loaded successfully", window)
	})

	clearConfigButton := widget.NewButton("Clear Config", func() {
		dialog.ShowConfirm("Clear Config", "Are you sure you want to clear all settings?", func(ok bool) {
			if ok {
				// Create empty config to clear all fields
				emptyConfig := &Config{
					Platform: "All",
					Android: struct {
						BuildType string `yaml:"build_type"`
					}{
						BuildType: "Release",
					},
				}
				updateUIFromConfig(emptyConfig, uiEntries)
			}
		}, window)
	})

	// Disable iOS fields if not on macOS
	if runtime.GOOS != "darwin" {
		platformRadio.Disable()              // Or just disable the 'iOS'/'All' option
		platformRadio.SetSelected("Android") // Default to Android if not macOS
		iosEnterpriseCheck.Disable()
		iosSchemeEntry.Disable()
		iosProjectNameEntry.Disable()
		appleIDEntry.Disable()
		teamIDEntry.Disable()
	}

	// Try to load existing config on startup
	configPath := filepath.Join(".", defaultConfig)
	if _, err := os.Stat(configPath); err == nil {
		config, err := LoadConfig(configPath)
		if err == nil {
			updateUIFromConfig(config, uiEntries)
		}
	}

	// Log Area
	logEntry = widget.NewMultiLineEntry()
	logEntry.Wrapping = fyne.TextWrapWord // Prevent wrapping for better log readability
	logEntry.SetMinRowsVisible(15)        // Show a good amount of log lines

	// Build Button
	buildButton := widget.NewButton("Run Build", nil) // OnTapped set later

	// --- Build Button Action ---
	buildButton.OnTapped = func() {
		logBuffer.Reset()    // Clear previous logs
		logEntry.SetText("") // Update UI
		buildButton.Disable()

		// --- Gather Config from UI ---
		config := getConfigFromUI(uiEntries)

		// Basic Validation
		if err := versionEntry.Validate(); err != nil {
			dialog.ShowError(fmt.Errorf("invalid build version: %w", err), window)
			buildButton.Enable()
			return
		}
		if config.Platform == "" {
			dialog.ShowError(fmt.Errorf("please select a platform"), window)
			buildButton.Enable()
			return
		}
		// Add more validation as needed (e.g., required fields for uploads)

		// --- Run Build in Goroutine ---
		go func() {
			// Ensure button is re-enabled when done
			defer buildButton.Enable()
			// Update log entry periodically or at the end
			// A simple way is to just update at the end, but better is periodic
			// For simplicity here, we update periodically via the LogWriter hook indirectly
			// by writing to the buffer which the main loop can check.
			// A more robust way involves channels or fyne.CurrentApp().QueueEvent.

			err := runBuildProcess(config, logWriter) // Pass the config and log writer

			if err != nil {
				// Show error dialog (must be called from main thread or via QueueEvent)
				// dialog.ShowError(err, window) // This might panic if called from goroutine
				log.Printf("Build Error: %v", err) // Log error to console as well
				// Append error to GUI log area safely
				fmt.Fprintf(logWriter, "\n\nBUILD FAILED: %v\n", err)
				logEntry.SetText(logBuffer.String()) // Refresh log view
			} else {
				fmt.Fprintf(logWriter, "\n\nBUILD SUCCEEDED!\n")
				logEntry.SetText(logBuffer.String()) // Refresh log view
				// dialog.ShowInformation("Success", "Build process completed successfully!", window) // Also needs main thread
			}
		}() // End of goroutine
	} // End of OnTapped

	// --- Layout ---
	// Use a Form for better label alignment
	form := widget.NewForm(
		widget.NewFormItem("Root Path", container.NewBorder(nil, nil, nil, rootPathButton, rootPathEntry)),
		widget.NewFormItem("Build Version*", versionEntry),
		widget.NewFormItem("Platform*", platformRadio),
		widget.NewFormItem("Options", container.NewHBox(skipUploadCheck, skipDepsCheck)),
	)

	androidSection := container.NewVBox(
		widget.NewLabel("Android Settings"),
		widget.NewForm(
			widget.NewFormItem("Build Type", androidBuildTypeEntry),
			widget.NewFormItem("Drive Folder ID", driveFolderEntry),
			widget.NewFormItem("Google Creds JSON", container.NewBorder(nil, nil, nil, googleCredsButton, googleCredsEntry)),
		),
	)

	iosSection := container.NewVBox(
		widget.NewLabel("iOS Settings"),
		widget.NewForm(
			widget.NewFormItem("", iosEnterpriseCheck), // No label for checkbox
			widget.NewFormItem("Scheme Override", iosSchemeEntry),
			widget.NewFormItem("Project Name Override", iosProjectNameEntry),
			widget.NewFormItem("Apple ID (Upload)", appleIDEntry),
			widget.NewFormItem("Team ID (Upload)", teamIDEntry),
		),
	)
	if runtime.GOOS != "darwin" {
		iosSection.Hide() // Hide iOS section if not on macOS
	}

	// Config Buttons container
	configButtons := container.NewHBox(
		saveConfigButton,
		loadConfigButton,
		clearConfigButton,
	)

	// Combine sections
	settings := container.NewVBox(
		configButtons,
		form,
		androidSection,
		iosSection,
	)

	logContainer = container.NewScroll(logEntry) // Make log area scrollable

	// Initialize the log writer with file output
	var err error
	logWriter, err = NewLogWriter()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to initialize logging: %w", err), window)
		return
	}
	defer logWriter.Close() // Ensure log file is closed when application exits

	// Main layout: Settings | Build Button | Logs
	content := container.NewBorder(
		settings,     // Top
		buildButton,  // Bottom
		nil,          // Left
		nil,          // Right
		logContainer, // Center
	)

	window.SetContent(content)
	window.ShowAndRun() // Blocks until window is closed
}
