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

var logBuffer bytes.Buffer // Buffer to store logs
var logWriter *LogWriter   // Our thread-safe writer wrapper
var logEntry *widget.Entry // The GUI text area for logs
var logContainer *container.Scroll
var lines []string

// LogWriter is a thread-safe writer for updating the GUI log
type LogWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

func NewLogWriter(w io.Writer) *LogWriter {
	return &LogWriter{writer: w}
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	// Add incoming data to the buffer
	writtenBytes, err := logBuffer.Write(p)
	if err != nil {
		// Return 0 bytes written and the error from buffer write
		return 0, err
	}

	linesAdded := false
	// Process lines from the buffer
	for {
		line, err := logBuffer.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Not a full line yet, put the partial line back into the buffer
				// (Note: ReadString consumes the data, so we need to write it back if it wasn't a full line)
				// However, since we check for EOF, the remaining partial line is already
				// left in the buffer by ReadString. So, we just break the loop.
				break
			}
			// Handle other potential read errors from the buffer if necessary
			// For simplicity, we'll just break here too. Consider logging this error.
			break // Or return 0, err depending on desired behavior
		}

		// We have a full line (including \n)
		linesAdded = true
		// Trim newline characters for storage
		processedLine := strings.TrimRight(line, "\r\n")

		// Add the line and enforce the maxLines limit
		lines = append(lines, processedLine)
		if len(lines) > 25 {
			// Remove the oldest line (from the beginning of the slice)
			lines = lines[1:]
		}
	}

	// If we added any new lines, update the Fyne widget
	if linesAdded {
		// Reconstruct the text content from the kept lines
		fullText := strings.Join(lines, "\n")

		// Schedule the UI update
		fyne.Do(func() {
			if logEntry != nil {
				logEntry.SetText(fullText) // Use SetText to replace content
			}
			if logContainer != nil {
				// Refresh and scroll (use delay if Refresh isn't enough - see previous answer)
				logContainer.Refresh()
			}
		})
	}

	// Return the number of bytes originally passed to Write, and nil error
	return writtenBytes, nil // Or return len(p), nil
}
