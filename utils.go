package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func isValidVersion(version string) bool { // Keep as is
	matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+$`, version)
	return matched
}

func calculateBuildNumberSimple(version string) (int, error) { // Keep as is
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid version format (expected X.Y.Z): %s", version)
	}
	num, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("invalid patch version part: %s", parts[2])
	}
	return num, nil
}

func getCurrentGitBranch(rootPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = rootPath // Set the working directory
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try fallback
		cmd = exec.Command("git", "branch", "--show-current")
		cmd.Dir = rootPath // Set the working directory for fallback
		output, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get git branch: %w - output: %s", err, string(output))
		}
	}
	return strings.TrimSpace(string(output)), nil
}

func findIOSWorkspaceAndScheme(config *Config) (workspace string, scheme string, err error) {
	iosDir := filepath.Join(config.RootPath, "ios")

	// --- Find Workspace ---
	if config.IOS.ProjectName != "" {
		// User override
		workspace = filepath.Join(iosDir, config.IOS.ProjectName+".xcworkspace")
		if _, statErr := os.Stat(workspace); statErr != nil {
			// Fallback to xcodeproj if workspace not found with override name
			xcodeproj := filepath.Join(iosDir, config.IOS.ProjectName+".xcodeproj")
			if _, projStatErr := os.Stat(xcodeproj); projStatErr == nil {
				fmt.Printf("Warning: Using project '%s' instead of workspace for override '%s'\n", xcodeproj, config.IOS.ProjectName)
				workspace = xcodeproj // Use project path if workspace doesn't exist
			} else {
				return "", "", fmt.Errorf("specified ios.project_name '%s' does not correspond to a .xcworkspace or .xcodeproj file in %s", config.IOS.ProjectName, iosDir)
			}
		}

	} else {
		// Auto-detect workspace
		files, readErr := os.ReadDir(iosDir)
		if readErr != nil {
			return "", "", fmt.Errorf("failed to read ios directory %s: %w", iosDir, readErr)
		}
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".xcworkspace") {
				workspace = filepath.Join(iosDir, file.Name())
				break // Take the first one found
			}
		}
		if workspace == "" {
			// If no workspace, look for project file
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".xcodeproj") {
					workspace = filepath.Join(iosDir, file.Name())
					fmt.Printf("Warning: No .xcworkspace found, using project '%s'\n", workspace)
					break
				}
			}
		}
		if workspace == "" {
			return "", "", errors.New("could not find .xcworkspace or .xcodeproj in ios directory")
		}
	}

	// --- Determine Scheme ---
	if config.IOS.Scheme != "" {
		// User override
		scheme = config.IOS.Scheme
	} else {
		// Auto-detect scheme (usually matches workspace/project name without extension)
		baseName := filepath.Base(workspace)
		if strings.HasSuffix(baseName, ".xcworkspace") {
			scheme = strings.TrimSuffix(baseName, ".xcworkspace")
		} else if strings.HasSuffix(baseName, ".xcodeproj") {
			scheme = strings.TrimSuffix(baseName, ".xcodeproj")
		} else {
			return "", "", fmt.Errorf("could not determine scheme from workspace/project path: %s", workspace)
		}
	}

	fmt.Printf("Using Workspace/Project: %s\n", workspace)
	fmt.Printf("Using Scheme: %s\n", scheme)
	return workspace, scheme, nil
}

func getGoogleTokenSource(credentialsPath string) (oauth2.TokenSource, error) {
	// No need for fallback here, path is validated before calling this function
	credentialsData, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file '%s': %w", credentialsPath, err)
	}

	// Use google.CredentialsFromJSON for broader credential type support (service account, user creds)
	creds, err := google.CredentialsFromJSON(context.Background(), credentialsData, googleDriveUploadScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials from JSON file '%s': %w", credentialsPath, err)
	}

	// The credentials object contains the TokenSource
	return creds.TokenSource, nil
}
