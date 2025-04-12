package main

import (
	"fmt"
	"log"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var guiApp fyne.App

func main() {
	guiApp = app.NewWithID("com.mps.rn_builder")
	window := guiApp.NewWindow("React Native Builder")
	window.Resize(fyne.NewSize(800, 600)) // Set a reasonable initial size

	// --- Create UI Widgets ---
	// Root Path
	rootPathEntry := widget.NewEntry()
	rootPathButton := widget.NewButton("Browse...", func() {
		dialog.NewFolderOpen(func(reader fyne.ListableURI, err error) {
			if err == nil && reader != nil {
				rootPathEntry.SetText(reader.Path())
			}
		}, window).Show()
	})

	// Build Version
	versionEntry := widget.NewEntry()
	versionEntry.Validator = func(s string) error {
		if !isValidVersion(s) { // Reuse your validation function
			return fmt.Errorf("invalid format (e.g., 1.2.3)")
		}
		return nil
	}

	// Platform
	platformRadio := widget.NewRadioGroup([]string{"All", "Android", "iOS"}, nil)
	platformRadio.SetSelected("All") // Default selection

	// Options
	skipUploadCheck := widget.NewCheck("Skip Uploads", nil)
	skipDepsCheck := widget.NewCheck("Skip Dependencies", nil)

	// Android Specific
	androidBuildTypeEntry := widget.NewEntry()
	androidBuildTypeEntry.SetText("Release") // Default
	driveFolderEntry := widget.NewEntry()
	googleCredsEntry := widget.NewEntry()
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
	iosSchemeEntry := widget.NewEntry()
	iosSchemeEntry.PlaceHolder = "Optional: Auto-detected"
	iosProjectNameEntry := widget.NewEntry()
	iosProjectNameEntry.PlaceHolder = "Optional: Auto-detected"
	appleIDEntry := widget.NewEntry()
	appleIDEntry.PlaceHolder = "Apple ID (email)"
	teamIDEntry := widget.NewEntry()
	teamIDEntry.PlaceHolder = "Apple Team ID (Optional)"

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

	// Log Area
	logEntry = widget.NewMultiLineEntry()
	logEntry.Wrapping = fyne.TextWrapWord // Prevent wrapping for better log readability
	logEntry.SetMinRowsVisible(15)        // Show a good amount of log lines
	logWriter = NewLogWriter(&logBuffer)

	// Build Button
	buildButton := widget.NewButton("Run Build", nil) // OnTapped set later

	// --- Build Button Action ---
	buildButton.OnTapped = func() {
		logBuffer.Reset()    // Clear previous logs
		logEntry.SetText("") // Update UI
		buildButton.Disable()

		// --- Gather Config from UI ---
		// Load base config from file? Or just use UI values? For simplicity, use UI.
		config := Config{
			RootPath:     rootPathEntry.Text,
			BuildVersion: versionEntry.Text,
			Platform:     platformRadio.Selected,
			SkipUpload:   skipUploadCheck.Checked,
			SkipDeps:     skipDepsCheck.Checked,
			Android: struct {
				BuildType string `yaml:"build_type"`
			}{
				BuildType: androidBuildTypeEntry.Text,
			},
			IOS: struct {
				Enterprise  bool   `yaml:"enterprise"`
				Scheme      string `yaml:"scheme"`
				ProjectName string `yaml:"project_name"`
			}{
				Enterprise:  iosEnterpriseCheck.Checked,
				Scheme:      iosSchemeEntry.Text,
				ProjectName: iosProjectNameEntry.Text,
			},
			DriveFolderID:     driveFolderEntry.Text,
			GoogleCredentials: googleCredsEntry.Text,
			AppleID:           appleIDEntry.Text,
			TeamID:            teamIDEntry.Text,
			// ReleaseChannel might be needed if prebuild uses it, add field if necessary
		}

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

	// Combine sections
	settings := container.NewVBox(form, androidSection, iosSection)
	logContainer = container.NewScroll(logEntry) // Make log area scrollable

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
