package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var logBuffer bytes.Buffer         // Buffer to store logs before processing into lines
var logWriter *LogWriter           // Our thread-safe writer wrapper
var logEntry *widget.Entry         // The GUI text area for logs
var logContainer *container.Scroll // The scroll container holding the logEntry
var lines []string                 // Slice to hold the lines currently displayed (limited size)
const maxLogLines = 25             // Define the maximum number of lines to keep

// LogWriter is a thread-safe writer for updating both file and GUI log
type LogWriter struct {
	mu       sync.Mutex
	file     *os.File
	logDir   string
	filename string
}

// NewLogWriter creates a new LogWriter that writes to both file and GUI
func NewLogWriter() (*LogWriter, error) {
	// Create logs directory if it doesn't exist
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("build_%s.log", timestamp)
	filepath := filepath.Join(logDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Initialize lines slice if it's nil
	if lines == nil {
		lines = make([]string, 0, maxLogLines)
	}

	return &LogWriter{
		file:     file,
		logDir:   logDir,
		filename: filename,
	}, nil
}

// Write processes incoming byte slices, writes to file and updates GUI
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	// Write to file first
	if n, err = lw.file.Write(p); err != nil {
		return n, fmt.Errorf("failed to write to log file: %w", err)
	}
	lw.file.Sync() // Ensure it's written to disk

	// Add incoming data to the GUI buffer
	originalLen := len(p)
	if _, err = logBuffer.Write(p); err != nil {
		return originalLen, nil // Continue even if GUI buffer fails
	}

	// Process full lines from the buffer for GUI
	linesAdded := false
	for {
		line, err := logBuffer.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}

		linesAdded = true
		processedLine := strings.TrimRight(line, "\r\n")

		lines = append(lines, processedLine)
		if len(lines) > maxLogLines {
			lines = lines[1:]
		}
	}

	// Update GUI if lines were added
	if linesAdded {
		fullText := strings.Join(lines, "\n")
		fyne.Do(func() {
			if logEntry != nil {
				logEntry.SetText(fullText)
			}
			if logContainer != nil {
				logContainer.Refresh()
				logContainer.ScrollToBottom()
			}
		})
	}

	return originalLen, nil
}

// Close closes the log file
func (lw *LogWriter) Close() error {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if lw.file != nil {
		return lw.file.Close()
	}
	return nil
}
