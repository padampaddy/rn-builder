package main

import (
	"bytes"
	"io"
	"strings"
	"sync"

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

// LogWriter is a thread-safe writer for updating the GUI log
type LogWriter struct {
	mu     sync.Mutex
	writer io.Writer // Note: This writer field isn't actually used in the current Write impl.
}

// NewLogWriter creates a new LogWriter.
// The provided io.Writer 'w' is stored but not directly used by the Write method below.
// The Write method interacts with the global logBuffer instead.
func NewLogWriter(w io.Writer) *LogWriter {
	// Initialize lines slice if it's nil
	if lines == nil {
		lines = make([]string, 0, maxLogLines)
	}
	return &LogWriter{writer: w}
}

// Write processes incoming byte slices, extracts lines, updates the limited line buffer,
// and schedules a GUI update to show only the latest lines and scroll to the bottom.
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	// Add incoming data to the buffer
	// We track the original number of bytes passed to Write
	originalLen := len(p)
	_, err = logBuffer.Write(p)
	if err != nil {
		// Return 0 bytes written (as per io.Writer expectation on error) and the error
		return 0, err
	}

	linesAdded := false
	// Process full lines from the buffer
	for {
		line, err := logBuffer.ReadString('\n')
		if err != nil {
			// If we encounter EOF, it means there's no complete line ending with '\n' left.
			// The partial line remains in the buffer for the next Write call.
			if err == io.EOF {
				break
			}
			// Handle other potential read errors from the buffer if necessary.
			// For now, we stop processing lines on error.
			// Consider logging this error `log.Printf("Error reading from log buffer: %v", err)`
			break // Stop processing lines on error
		}

		// We have a full line (including \n)
		linesAdded = true
		// Trim trailing newline characters for storage
		processedLine := strings.TrimRight(line, "\r\n")

		// Add the new line and enforce the maxLogLines limit
		lines = append(lines, processedLine)
		if len(lines) > maxLogLines {
			// Remove the oldest line (from the beginning of the slice)
			// This keeps the slice size at maxLogLines
			lines = lines[1:]
		}
	}

	// If we added any new lines, update the Fyne widget
	if linesAdded {
		// Reconstruct the text content from the currently kept lines
		fullText := strings.Join(lines, "\n")

		// Schedule the UI update to run on the main Fyne goroutine
		fyne.Do(func() {
			if logEntry != nil {
				logEntry.SetText(fullText) // Replace the entire content
				// logEntry.CursorColumn = 0 // Optionally move cursor if needed
				// logEntry.CursorRow = len(lines) // Not reliable for scrolling
			}
			if logContainer != nil {
				// Refresh the container to ensure layout is updated
				logContainer.Refresh()
				// *** Add this line to scroll to the bottom ***
				logContainer.ScrollToBottom()
			}
		})
	}

	// Per io.Writer contract, return the number of bytes *accepted* from p.
	// Since logBuffer.Write consumes all of p unless it returns an error,
	// we return the original length if the buffer write was successful.
	return originalLen, nil
}

// --- Helper function to initialize the log system (example) ---
// You would call this during your app setup
func setupLogViewer(logDisplay *widget.Entry, scrollArea *container.Scroll) io.Writer {
	logEntry = logDisplay
	logContainer = scrollArea

	// Make the log entry read-only and multi-line
	logEntry.MultiLine = true
	logEntry.Disable()
	logEntry.Wrapping = fyne.TextWrapWord // Or TextWrapOff, TextWrapBreak
	// logEntry.Disable() // Or use logEntry.ReadOnly = true in newer Fyne versions if available

	// Create the log writer (passing logBuffer, though it's not directly used by NewLogWriter)
	logWriter = NewLogWriter(&logBuffer) // The writer passed here isn't used by Write

	// Return the logWriter so other parts of the app can write to it
	// Example: log.SetOutput(logWriter)
	return logWriter
}
