package main

const (
	expoCli                      = "npx expo"                                                  // Use npx to ensure local or latest expo-cli
	altoolPath                   = "/Applications/Xcode.app/Contents/Developer/usr/bin/altool" // Path for altool (used for TestFlight upload)
	iosOutputDir                 = "dist/ios"                                                  // Changed output dir for local builds
	androidOutput                = "dist/android"                                              // Changed output dir for local builds
	defaultConfig                = "rn-builder.yaml"
	googleDriveUploadScope       = "https://www.googleapis.com/auth/drive.file"
	googleDriveUploadURL         = "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart"
	googleDriveMetadataURL       = "https://www.googleapis.com/drive/v3/files"
	exportOptionsAppStorePlist   = "ExportOptionsAppStore.plist"   // Assumed name for App Store plist
	exportOptionsEnterprisePlist = "ExportOptionsEnterprise.plist" // Assumed name for Enterprise plist
)
