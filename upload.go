package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
)

type GoogleDriveFile struct {
	Name     string   `json:"name"`
	MimeType string   `json:"mimeType"`
	Parents  []string `json:"parents,omitempty"`
}

func uploadToTestFlightGUI(config Config, isMainBranch bool, ipaPath string, logOutput io.Writer) error {
	fmt.Fprintln(logOutput, "Uploading IPA to TestFlight/App Store Connect...")

	// Check if IPA file exists
	if _, err := os.Stat(ipaPath); os.IsNotExist(err) {
		return fmt.Errorf("IPA file not found for upload: %s", ipaPath)
	}

	// Check if altool is available
	altoolCmd := "xcrun altool" // Use xcrun altool
	if _, err := exec.LookPath("xcrun"); err != nil {
		fmt.Fprintln(logOutput, "Warning: 'xcrun' not found in PATH. Falling back to direct altool path.")
		altoolCmd = altoolPath // Fallback to hardcoded path
		if _, err := os.Stat(altoolCmd); os.IsNotExist(err) {
			return fmt.Errorf("altool/xcrun not found, Xcode Command Line Tools might be missing or not configured correctly, cannot upload")
		}
	}

	appleID := config.AppleID
	if appleID == "" {
		appleID = os.Getenv("APPLE_ID")
		if appleID == "" {
			return errors.New("apple ID not provided in config (apple_id) or APPLE_ID environment variable")
		}
	}

	// App-Specific Password handling
	passwordEnv := os.Getenv("APP_STORE_CONNECT_PASSWORD")
	passwordArg := "@env:APP_STORE_CONNECT_PASSWORD" // Default to using env var
	if passwordEnv == "" {
		fmt.Fprintln(logOutput, "Using '@keychain:AC_PASSWORD' for App Store Connect password. Ensure it is set in Keychain Access.")
		passwordArg = "@keychain:AC_PASSWORD"
	} else {
		fmt.Fprintf(logOutput, "Using environment variable '%s' for App Store Connect password.\n", "APP_STORE_CONNECT_PASSWORD")
	}

	uploadArgs := []string{
		"--upload-app",
		"-t", "ios", // type ios
		"-f", ipaPath, // file
		"-u", appleID, // username
		"-p", passwordArg, // password (@keychain:item or @env:VAR)
	}

	// Add ASC Provider (Team ID) if available
	teamID := config.TeamID
	if teamID == "" {
		teamID = os.Getenv("TEAM_ID")
	}
	if teamID != "" {
		fmt.Fprintf(logOutput, "Using Team ID (ASC Provider): %s\n", teamID)
		uploadArgs = append(uploadArgs, "--asc-provider", teamID)
	} else {
		fmt.Fprintln(logOutput, "Note: No Team ID (asc-provider) provided. Upload will use the default associated with the Apple ID.")
	}

	fmt.Fprintln(logOutput, "Starting upload command (this might take a while)...")
	if err := runCmd(logOutput, false, "", altoolCmd, uploadArgs...); err != nil {
		// Provide more helpful error message for common auth issues
		if strings.Contains(err.Error(), "Authentication failed") || strings.Contains(err.Error(), "status 401") {
			return fmt.Errorf("TestFlight upload authentication failed. Check Apple ID, password/keychain item (%s), and potentially 2FA requirements: %w", passwordArg, err)
		}
		return fmt.Errorf("TestFlight upload command failed: %w", err)
	}

	fmt.Fprintln(logOutput, "IPA uploaded to TestFlight/App Store Connect successfully (processing may continue on Apple's side)")
	return nil
}

func uploadToGoogleDriveWithAPIGUI(config Config, apkPath string, logOutput io.Writer) error {
	fmt.Fprintln(logOutput, "Uploading APK to Google Drive using API...")

	// Check if APK exists
	if _, err := os.Stat(apkPath); os.IsNotExist(err) {
		return fmt.Errorf("APK file not found for upload: %s", apkPath)
	}

	// Validate credentials path early
	credentialsPath := config.GoogleCredentials
	if credentialsPath == "" {
		credentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if credentialsPath == "" {
			return errors.New("google credentials path not specified in config (google_credentials) or GOOGLE_APPLICATION_CREDENTIALS environment variable")
		}
	}
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		return fmt.Errorf("google credentials file not found at: %s", credentialsPath)
	}

	// Get OAuth2 token source
	tokenSource, err := getGoogleTokenSource(credentialsPath)
	if err != nil {
		return fmt.Errorf("failed to get Google token source: %w", err)
	}

	// Create HTTP client with OAuth2
	client := oauth2.NewClient(context.Background(), tokenSource)

	// Open the file
	file, err := os.Open(apkPath)
	if err != nil {
		return fmt.Errorf("failed to open APK file '%s': %w", apkPath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info for '%s': %w", apkPath, err)
	}
	fileSize := fileInfo.Size()
	fmt.Fprintf(logOutput, "Uploading file: %s (%d bytes)\n", filepath.Base(apkPath), fileSize)

	// Create a pipe to stream the multipart request
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	uploadErrChan := make(chan error, 1) // Channel to capture errors from goroutine

	// Start a goroutine to write the multipart data
	go func() {
		defer pw.Close()     // Ensure pipe writer is closed
		defer writer.Close() // Ensure multipart writer finishes

		var writeErr error // Variable to store error within goroutine

		defer func() { // Send error back through channel
			uploadErrChan <- writeErr
		}()

		// Create metadata part
		metadata := GoogleDriveFile{
			Name:     filepath.Base(apkPath), // Use the actual filename
			MimeType: "application/vnd.android.package-archive",
			Parents:  []string{config.DriveFolderID},
		}

		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			writeErr = fmt.Errorf("failed to marshal metadata: %w", err)
			pw.CloseWithError(writeErr) // Close pipe with error
			return
		}

		// Create metadata header
		metadataHeader := textproto.MIMEHeader{}
		metadataHeader.Set("Content-Type", "application/json; charset=UTF-8")
		part, err := writer.CreatePart(metadataHeader)
		if err != nil {
			writeErr = fmt.Errorf("failed to create metadata part: %w", err)
			pw.CloseWithError(writeErr)
			return
		}
		if _, err := part.Write(metadataJSON); err != nil {
			writeErr = fmt.Errorf("failed to write metadata JSON: %w", err)
			pw.CloseWithError(writeErr)
			return
		}

		// Create file part
		fileHeader := textproto.MIMEHeader{}
		fileHeader.Set("Content-Type", "application/vnd.android.package-archive")
		part, err = writer.CreatePart(fileHeader)
		if err != nil {
			writeErr = fmt.Errorf("failed to create file part: %w", err)
			pw.CloseWithError(writeErr)
			return
		}

		// Copy file data with progress indication
		fmt.Fprintln(logOutput, "Starting file data copy to upload stream...")
		copiedBytes, err := io.Copy(part, file)
		if err != nil {
			writeErr = fmt.Errorf("failed to copy file data to upload stream: %w", err)
			pw.CloseWithError(writeErr)
			return
		}
		fmt.Fprintf(logOutput, "Finished copying %d bytes to upload stream.\n", copiedBytes)

	}() // End of goroutine

	// Create the request
	req, err := http.NewRequestWithContext(context.Background(), "POST", googleDriveUploadURL, pr)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.ContentLength = -1 // Let the client handle streaming length or chunking

	// Make the request
	fmt.Fprintln(logOutput, "Sending upload request to Google Drive API...")
	resp, err := client.Do(req)
	if err != nil {
		// Check error from the goroutine writing the pipe *before* blaming the client.Do call
		writerErr := <-uploadErrChan
		if writerErr != nil {
			return fmt.Errorf("error occurred during upload data preparation: %w", writerErr)
		}
		// If no writer error, then the network request itself failed
		return fmt.Errorf("failed to execute Google Drive upload request: %w", err)
	}
	defer resp.Body.Close()

	// Check error from goroutine even if request succeeded
	writerErr := <-uploadErrChan
	if writerErr != nil && resp.StatusCode == http.StatusOK {
		// This case is unlikely but possible if the server responded OK before reading full body
		fmt.Fprintf(logOutput, "Warning: Google Drive API returned OK, but data writing encountered an error: %v\n", writerErr)
	} else if writerErr != nil {
		// If writer failed and response code is also error, report writer error primarily
		return fmt.Errorf("error occurred during upload data preparation: %w", writerErr)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("google Drive upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	fmt.Fprintln(logOutput, "Google Drive API request successful.")
	fmt.Fprintln(logOutput, "APK uploaded to Google Drive successfully.")
	return nil
}
