package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func updateEnvironmentConstant(config Config, branch string, logOutput io.Writer) error {
	constantsFilePath := filepath.Join(config.RootPath, "src", "utils", "constants.js")
	fmt.Fprintf(logOutput, "Updating environment constant in %s based on branch: %s\n", constantsFilePath, branch)

	// Read the constants file
	content, err := os.ReadFile(constantsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read constants file: %w", err)
	}

	// Determine the environment based on the branch
	var environment string
	switch branch {
	case "main":
		environment = "PROD"
	case "staging":
		environment = "STAGING"
	default:
		environment = "DEV"
	}

	// Replace the ENVIRONMENT constant
	updatedContent := strings.ReplaceAll(
		string(content),
		`const ENVIRONMENT = "DEV";`, // Default assumption, adjust if needed
		fmt.Sprintf(`const ENVIRONMENT = "%s";`, environment),
	)

	// Write the updated content back to the file
	if err := os.WriteFile(constantsFilePath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated constants file: %w", err)
	}

	fmt.Fprintf(logOutput, "Environment constant updated to %s successfully.\n", environment)
	return nil
}

func runBuildProcess(config Config, logOutput io.Writer) error {
	fmt.Fprintf(logOutput, "Starting build process for version %s...\n", config.BuildVersion)

	// Calculate build number (reuse existing function)
	buildNumber, err := calculateBuildNumberSimple(config.BuildVersion)
	if err != nil {
		fmt.Fprintf(logOutput, "Error calculating build number: %v\n", err)
		return fmt.Errorf("error calculating build number: %w", err)
	}
	fmt.Fprintf(logOutput, "Using Build Number: %d\n", buildNumber)

	// Check current branch (optional, reuse function)
	currentBranch, err := getCurrentGitBranch(config.RootPath)
	if err != nil {
		fmt.Fprintf(logOutput, "Warning: could not determine git branch: %v\n", err)
		currentBranch = "unknown"
	}
	isMainBranch := currentBranch == "main"
	fmt.Fprintf(logOutput, "Git Branch: %s (Is Main: %t)\n", currentBranch, isMainBranch)

	// Update environment constant
	if err := updateEnvironmentConstant(config, currentBranch, logOutput); err != nil {
		return fmt.Errorf("error updating environment constant: %w", err)
	}

	// Install dependencies if not skipped
	if !config.SkipDeps {
		fmt.Fprintf(logOutput, "Running dependency installation...\n")
		// Pass logOutput to installDependencies if it needs logging
		if err := installDependenciesGUI(config, logOutput); err != nil { // Assume modified installDependenciesGUI
			return fmt.Errorf("error installing dependencies: %w", err)
		}
		fmt.Fprintf(logOutput, "Dependency installation finished.\n")
	} else {
		fmt.Fprintf(logOutput, "Skipping dependency installation.\n")
	}

	var androidArtifactPath string
	var iosArtifactPath string
	var buildErr error

	// Process builds based on platform
	platformLower := strings.ToLower(config.Platform)

	if platformLower == "all" || platformLower == "android" {
		// Assume buildAndroid is modified to accept logOutput io.Writer
		androidArtifactPath, buildErr = buildAndroidGUI(config, buildNumber, isMainBranch, logOutput)
		if buildErr != nil {
			return fmt.Errorf("android build failed: %w", buildErr)
		}
	}

	if platformLower == "all" || platformLower == "ios" {
		if runtime.GOOS != "darwin" {
			fmt.Fprintf(logOutput, "Skipping iOS build: requires macOS\n")
		} else {
			// Assume buildIOS is modified to accept logOutput io.Writer
			iosArtifactPath, buildErr = buildIOSGUI(config, buildNumber, isMainBranch, logOutput)
			if buildErr != nil {
				return fmt.Errorf("ios build failed: %w", buildErr)
			}
		}
	}

	if platformLower != "all" && platformLower != "android" && platformLower != "ios" {
		return fmt.Errorf("invalid platform specified: %s", config.Platform)
	}

	// Handle uploads if not skipped
	if !config.SkipUpload {
		fmt.Fprintf(logOutput, "Handling uploads...\n")
		if androidArtifactPath != "" {
			if config.DriveFolderID == "" || config.GoogleCredentials == "" {
				fmt.Fprintf(logOutput, "Skipping Google Drive upload: Drive Folder ID or Google Credentials Path not provided.\n")
			} else {
				// Assume uploadToGoogleDriveWithAPI is modified to accept logOutput
				if err := uploadToGoogleDriveWithAPIGUI(config, androidArtifactPath, logOutput); err != nil {
					return fmt.Errorf("google drive upload failed: %w", err)
				}
			}
		}

		if iosArtifactPath != "" && runtime.GOOS == "darwin" {
			// Assume uploadToTestFlight is modified to accept logOutput
			if err := uploadToTestFlightGUI(config, isMainBranch, iosArtifactPath, logOutput); err != nil {
				return fmt.Errorf("test flight upload failed: %w", err)
			}
		}
	} else {
		fmt.Fprintf(logOutput, "Skipping uploads.\n")
	}

	fmt.Fprintf(logOutput, "Build process seems complete.\n")
	return nil // Success
}

// Modify installDependencies to accept logOutput
func installDependenciesGUI(config Config, logOutput io.Writer) error {
	fmt.Fprintln(logOutput, "Installing npm dependencies...")
	if err := runCmd(logOutput, true, config.RootPath, "npm", "install"); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	if runtime.GOOS == "darwin" {
		fmt.Fprintln(logOutput, "Installing CocoaPods dependencies...")
		iosDir := filepath.Join(config.RootPath, "ios")
		// Basic check if dir exists
		if _, err := os.Stat(iosDir); os.IsNotExist(err) {
			fmt.Fprintln(logOutput, "Skipping pod install: 'ios' directory not found (run prebuild first?)")
			return nil
		}
		// Add podfile check etc. if needed

		podCmd := "pod"
		podArgs := []string{"install"}
		// Add bundler check if desired

		if err := runCmd(logOutput, true, iosDir, podCmd, podArgs...); err != nil {
			return fmt.Errorf("pod install failed: %w", err)
		}
	}
	return nil
}

// Modify buildAndroid to accept logOutput and use runCmd properly
func buildAndroidGUI(config Config, buildNumber int, isMainBranch bool, logOutput io.Writer) (string, error) {
	fmt.Fprintln(logOutput, "Building Android app using prebuild and Gradle...")
	// --- Setup ---
	if err := os.MkdirAll(androidOutput, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir %s: %w", androidOutput, err)
	}

	// --- Prebuild ---
	fmt.Fprintln(logOutput, "Running expo prebuild...")
	expoCmd := "npx"
	prebuildArgs := []string{"expo", "prebuild", "--platform", "android", "--no-install"}
	if err := runCmd(logOutput, true, config.RootPath, expoCmd, prebuildArgs...); err != nil {
		return "", fmt.Errorf("expo prebuild failed: %w", err)
	}

	// --- Update Build Number and Version Name in build.gradle ---
	fmt.Fprintln(logOutput, "Updating build number and version name in build.gradle...")
	buildGradlePath := filepath.Join(config.RootPath, "android", "app", "build.gradle")
	buildGradleContent, err := os.ReadFile(buildGradlePath)
	if err != nil {
		return "", fmt.Errorf("failed to read build.gradle: %w", err)
	}

	updatedContent := strings.ReplaceAll(
		string(buildGradleContent),
		"versionCode 1", // Replace this with a more robust regex or logic if needed
		fmt.Sprintf("versionCode %d", buildNumber),
	)

	updatedContent = strings.ReplaceAll(
		updatedContent,
		`versionName "1.0"`, // Replace this with a more robust regex or logic if needed
		fmt.Sprintf(`versionName "%s"`, config.BuildVersion),
	)

	if err := os.WriteFile(buildGradlePath, []byte(updatedContent), 0644); err != nil {
		return "", fmt.Errorf("failed to update build.gradle: %w", err)
	}
	fmt.Fprintln(logOutput, "Build number and version name updated successfully.")

	// --- Gradle Build ---
	gradleTask := fmt.Sprintf("assemble%s", config.Android.BuildType)
	fmt.Fprintf(logOutput, "Running Gradle task: %s\n", gradleTask)
	androidProjectDir := filepath.Join(config.RootPath, "android")
	gradlewPath := "./gradlew"
	if runtime.GOOS == "windows" {
		gradlewPath = "./gradlew.bat"
	}
	// Check gradlew exists
	if _, err := os.Stat(filepath.Join(androidProjectDir, "gradlew")); os.IsNotExist(err) {
		return "", fmt.Errorf("gradlew script not found")
	}

	if err := runCmd(logOutput, true, androidProjectDir, gradlewPath, gradleTask); err != nil {
		return "", fmt.Errorf("gradle build failed (%s): %w", gradleTask, err)
	}

	// --- Locate and Move APK (Simplified - copy logic from original) ---
	apkBuildTypeDir := strings.ToLower(config.Android.BuildType)
	apkOutputDir := filepath.Join(androidProjectDir, "app", "build", "outputs", "apk", apkBuildTypeDir)
	apkFiles, err := filepath.Glob(filepath.Join(apkOutputDir, "*.apk"))
	if err != nil || len(apkFiles) == 0 {
		return "", fmt.Errorf("no APK found in %s", apkOutputDir)
	}
	apkSourcePath := apkFiles[0]
	fmt.Fprintf(logOutput, "Found generated APK: %s\n", apkSourcePath)

	profileSuffix := strings.ToLower(config.Android.BuildType)
	if isMainBranch && profileSuffix == "release" {
		profileSuffix = "production"
	}
	destFileName := fmt.Sprintf("app-%s-%d-%s.apk", config.BuildVersion, buildNumber, profileSuffix)
	destPath := filepath.Join(androidOutput, destFileName)

	fmt.Fprintf(logOutput, "Moving APK to %s\n", destPath)
	if err := os.Rename(apkSourcePath, destPath); err != nil {
		// Add copy+delete fallback if needed
		return "", fmt.Errorf("failed to move APK: %w", err)
	}

	fmt.Fprintf(logOutput, "Android build complete: %s\n", destPath)
	return destPath, nil
}

// Modify buildIOS similarly...
func buildIOSGUI(config Config, buildNumber int, isMainBranch bool, logOutput io.Writer) (string, error) {
	fmt.Fprintln(logOutput, "Building iOS app using prebuild and xcodebuild...")
	if runtime.GOOS != "darwin" {
		return "", errors.New("iOS builds require macOS")
	}

	// Setup output dir
	if err := os.MkdirAll(iosOutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create dir %s: %w", iosOutputDir, err)
	}

	// Prebuild
	fmt.Fprintln(logOutput, "Running expo prebuild...")
	expoCmd := "npx"
	prebuildArgs := []string{"expo", "prebuild", "--platform", "ios", "--no-install"}
	if err := runCmd(logOutput, true, config.RootPath, expoCmd, prebuildArgs...); err != nil {
		return "", fmt.Errorf("expo prebuild failed: %w", err)
	}

	// Update build number, version, app name, and package ID in Info.plist
	fmt.Fprintln(logOutput, "Updating build number, version, app name, and package ID in Info.plist...")
	infoPlistPath := filepath.Join(config.RootPath, "ios", config.IOS.ProjectName, "Info.plist")
	infoPlistContent, err := os.ReadFile(infoPlistPath)
	if err != nil {
		return "", fmt.Errorf("failed to read Info.plist: %w", err)
	}

	// Determine app name and package ID based on branch
	var appName, packageID string
	currentBranch, err := getCurrentGitBranch(config.RootPath)
	if err != nil {
		return "", fmt.Errorf("failed to determine git branch: %w", err)
	}

	switch currentBranch {
	case "main":
		appName = "Sunflow Installer"
		packageID = "com.sunflow.installer.app"
	case "staging":
		appName = "Staging Sunflow Installer"
		packageID = "com.sunflow.installer.app.ios.testing"
	case "develop":
		appName = "Dev Sunflow Installer"
		packageID = "com.sunflow.installer.app.dev"
	default:
		appName = "Dev Sunflow Installer"
		packageID = "com.sunflow.installer.app.dev"
	}

	reCFBundleVersion := regexp.MustCompile(`<key>CFBundleVersion</key>\s*<string>.*?</string>`)
	updatedContent := reCFBundleVersion.ReplaceAllString(
		string(infoPlistContent),
		fmt.Sprintf("<key>CFBundleVersion</key>\n\t<string>%d</string>", buildNumber),
	)

	reCFBundleShortVersion := regexp.MustCompile(`<key>CFBundleShortVersionString</key>\s*<string>.*?</string>`)
	updatedContent = reCFBundleShortVersion.ReplaceAllString(
		updatedContent,
		fmt.Sprintf("<key>CFBundleShortVersionString</key>\n\t<string>%s</string>", config.BuildVersion),
	)

	reCFBundleName := regexp.MustCompile(`<key>CFBundleName</key>\s*<string>.*?</string>`)
	updatedContent = reCFBundleName.ReplaceAllString(
		updatedContent,
		fmt.Sprintf("<key>CFBundleName</key>\n\t<string>%s</string>", appName),
	)

	reCFBundleIdentifier := regexp.MustCompile(`<key>CFBundleIdentifier</key>\s*<string>.*?</string>`)
	updatedContent = reCFBundleIdentifier.ReplaceAllString(
		updatedContent,
		fmt.Sprintf("<key>CFBundleIdentifier</key>\n\t<string>%s</string>", packageID),
	)

	if err := os.WriteFile(infoPlistPath, []byte(updatedContent), 0644); err != nil {
		return "", fmt.Errorf("failed to update Info.plist: %w", err)
	}
	fmt.Fprintln(logOutput, "Build number, version, app name, and package ID updated successfully.")

	workspace, scheme, err := findIOSWorkspaceAndScheme(&config)
	if err != nil {
		return "", fmt.Errorf("ios workspace error: %w", err)
	}

	// Archive
	fmt.Fprintln(logOutput, "Running xcodebuild archive...")
	archiveName := fmt.Sprintf("%s.xcarchive", scheme)
	archivePath := filepath.Join(config.RootPath, iosOutputDir, archiveName)
	_ = os.RemoveAll(archivePath) // Clean previous

	archiveArgs := []string{
		"workspace",
		workspace,
		"-scheme", scheme,
		"-configuration", "Release",
		"-sdk", "iphoneos",
		"-archivePath", archivePath,
		"archive",
	}
	if teamID := config.TeamID; teamID != "" {
		archiveArgs = append(archiveArgs, fmt.Sprintf("DEVELOPMENT_TEAM=%s", teamID))
	}
	// Use xcodebuild directly, not via shell, as it's usually in PATH
	if err := runCmd(logOutput, true, config.RootPath, "xcodebuild", archiveArgs...); err != nil {
		return "", fmt.Errorf("xcodebuild archive failed: %w", err)
	}

	// Export Archive
	fmt.Fprintln(logOutput, "Running xcodebuild exportArchive...")
	exportDir := filepath.Join(config.RootPath, iosOutputDir, "export")
	plistName := exportOptionsAppStorePlist
	exportMethod := "app-store"
	if config.IOS.Enterprise {
		plistName = exportOptionsEnterprisePlist
		exportMethod = "enterprise"
	}
	plistPath := filepath.Join(".", plistName)
	if _, err := os.Stat(plistPath); err != nil {
		return "", fmt.Errorf("exportOptionsPlist '%s' not found for %s export", plistPath, exportMethod)
	}
	_ = os.RemoveAll(exportDir) // Clean previous

	exportArgs := []string{
		"-exportArchive",
		"-archivePath", archivePath,
		"-exportPath", exportDir,
		"-exportOptionsPlist", plistPath,
	}
	if err := runCmd(logOutput, true, config.RootPath, "xcodebuild", exportArgs...); err != nil {
		return "", fmt.Errorf("xcodebuild exportArchive failed: %w", err)
	}

	// Locate and Move IPA (Simplified)
	ipaPattern := filepath.Join(exportDir, "*.ipa")
	ipaFiles, err := filepath.Glob(ipaPattern)
	if err != nil || len(ipaFiles) == 0 {
		return "", fmt.Errorf("no IPA found in export dir: %s", exportDir)
	}
	ipaSourcePath := ipaFiles[0]
	fmt.Fprintf(logOutput, "Found generated IPA: %s\n", ipaSourcePath)

	profileSuffix := "appstore"
	if config.IOS.Enterprise {
		profileSuffix = "enterprise"
	}
	destFileName := fmt.Sprintf("%s-%s-%d-%s.ipa", scheme, config.BuildVersion, buildNumber, profileSuffix)
	destPath := filepath.Join(iosOutputDir, destFileName)

	fmt.Fprintf(logOutput, "Moving IPA to %s\n", destPath)
	if err := os.Rename(ipaSourcePath, destPath); err != nil {
		return "", fmt.Errorf("failed to move IPA: %w", err)
	}

	_ = os.RemoveAll(archivePath)
	_ = os.RemoveAll(exportDir)

	fmt.Fprintf(logOutput, "iOS build complete: %s\n", destPath)
	return destPath, nil
}
