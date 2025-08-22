package logging

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DailyFileConfig holds configuration for the daily file writer
type DailyFileConfig struct {
	Directory  string // Directory to store log files
	MaxAge     int    // Maximum number of days to retain log files
	MaxBackups int    // Maximum number of backup files to keep
	Compress   bool   // Whether to compress rotated files
	LocalTime  bool   // Whether to use local time for file names
}

// DailyFileWriter handles daily log file creation and rotation
type DailyFileWriter struct {
	directory    string
	filename     string
	currentFile  *os.File
	currentDate  string
	maxAge       int
	maxBackups   int
	compress     bool
	localTime    bool
	mu           sync.Mutex
}

// NewDailyFileWriter creates a new daily file writer with the given configuration
func NewDailyFileWriter(config DailyFileConfig) (*DailyFileWriter, error) {
	// Ensure directory exists
	if err := os.MkdirAll(config.Directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	dfw := &DailyFileWriter{
		directory:  config.Directory,
		maxAge:     config.MaxAge,
		maxBackups: config.MaxBackups,
		compress:   config.Compress,
		localTime:  config.LocalTime,
	}

	// Initialize with current date and open initial file
	if err := dfw.openFile(); err != nil {
		return nil, fmt.Errorf("failed to open initial log file: %w", err)
	}

	return dfw, nil
}

// Write implements io.Writer interface with automatic rotation
func (dfw *DailyFileWriter) Write(p []byte) (n int, err error) {
	dfw.mu.Lock()
	defer dfw.mu.Unlock()

	// Check if we need to rotate
	if dfw.shouldRotate() {
		if err := dfw.rotate(); err != nil {
			return 0, fmt.Errorf("failed to rotate log file: %w", err)
		}
	}

	// Write to current file
	if dfw.currentFile == nil {
		return 0, fmt.Errorf("no current log file available")
	}

	return dfw.currentFile.Write(p)
}

// shouldRotate checks if log rotation is needed based on date change
func (dfw *DailyFileWriter) shouldRotate() bool {
	currentDate := dfw.getCurrentDate()
	return dfw.currentDate != currentDate
}

// getCurrentDate returns the current date string for file naming
func (dfw *DailyFileWriter) getCurrentDate() string {
	now := time.Now()
	if !dfw.localTime {
		now = now.UTC()
	}
	return now.Format("2006-01-02")
}

// getLogFileName returns the log file name for a given date
func (dfw *DailyFileWriter) getLogFileName(date string) string {
	return filepath.Join(dfw.directory, fmt.Sprintf("app-%s.log", date))
}

// openFile opens the log file for the current date
func (dfw *DailyFileWriter) openFile() error {
	currentDate := dfw.getCurrentDate()
	filename := dfw.getLogFileName(currentDate)

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", filename, err)
	}

	dfw.currentFile = file
	dfw.currentDate = currentDate
	dfw.filename = filename

	return nil
}

// rotate performs log file rotation
func (dfw *DailyFileWriter) rotate() error {
	// Store old file info for compression
	var oldFilename string
	if dfw.currentFile != nil {
		oldFilename = dfw.filename
		
		// Close current file
		if err := dfw.currentFile.Close(); err != nil {
			return fmt.Errorf("failed to close current log file: %w", err)
		}
		dfw.currentFile = nil
	}

	// Open new file for current date
	if err := dfw.openFile(); err != nil {
		return fmt.Errorf("failed to open new log file: %w", err)
	}

	// Compress old file if configured and it exists
	if oldFilename != "" && dfw.compress {
		go func() {
			if err := dfw.compressFile(oldFilename); err != nil {
				// Log error but don't fail rotation
				fmt.Printf("Warning: failed to compress log file %s: %v\n", oldFilename, err)
			}
		}()
	}

	// Run cleanup in background
	go func() {
		if err := dfw.cleanup(); err != nil {
			// Log error but don't fail rotation
			fmt.Printf("Warning: failed to cleanup old log files: %v\n", err)
		}
	}()

	return nil
}

// Rotate manually triggers log rotation (useful for testing)
func (dfw *DailyFileWriter) Rotate() error {
	dfw.mu.Lock()
	defer dfw.mu.Unlock()
	return dfw.rotate()
}

// Close closes the current log file
func (dfw *DailyFileWriter) Close() error {
	dfw.mu.Lock()
	defer dfw.mu.Unlock()

	if dfw.currentFile != nil {
		err := dfw.currentFile.Close()
		dfw.currentFile = nil
		return err
	}
	return nil
}

// Sync flushes any buffered data to the underlying file
func (dfw *DailyFileWriter) Sync() error {
	dfw.mu.Lock()
	defer dfw.mu.Unlock()

	if dfw.currentFile != nil {
		return dfw.currentFile.Sync()
	}
	return nil
}

// GetCurrentFileName returns the current log file name
func (dfw *DailyFileWriter) GetCurrentFileName() string {
	dfw.mu.Lock()
	defer dfw.mu.Unlock()
	return dfw.filename
}

// compressFile compresses a log file using gzip
func (dfw *DailyFileWriter) compressFile(filename string) error {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to compress
	}

	// Open source file
	src, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// Create compressed file
	compressedFilename := filename + ".gz"
	dst, err := os.Create(compressedFilename)
	if err != nil {
		return fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer dst.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(dst)
	defer gzWriter.Close()

	// Copy data
	if _, err := io.Copy(gzWriter, src); err != nil {
		// Clean up partial compressed file
		os.Remove(compressedFilename)
		return fmt.Errorf("failed to compress file: %w", err)
	}

	// Close gzip writer to flush data
	if err := gzWriter.Close(); err != nil {
		os.Remove(compressedFilename)
		return fmt.Errorf("failed to close compressed file: %w", err)
	}

	// Remove original file after successful compression
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to remove original file after compression: %w", err)
	}

	return nil
}

// cleanup removes old log files based on retention policies
func (dfw *DailyFileWriter) cleanup() error {
	// Get all log files in directory
	pattern := filepath.Join(dfw.directory, "app-*.log*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list log files: %w", err)
	}

	if len(files) == 0 {
		return nil // No files to clean up
	}

	// Get file info for all log files
	type fileInfo struct {
		path    string
		modTime time.Time
		isGzip  bool
	}

	var logFiles []fileInfo
	for _, file := range files {
		// Skip current log file
		if file == dfw.filename {
			continue
		}

		stat, err := os.Stat(file)
		if err != nil {
			continue // Skip files we can't stat
		}

		logFiles = append(logFiles, fileInfo{
			path:    file,
			modTime: stat.ModTime(),
			isGzip:  strings.HasSuffix(file, ".gz"),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].modTime.After(logFiles[j].modTime)
	})

	// Apply cleanup policies
	now := time.Now()
	var filesToRemove []string

	for i, file := range logFiles {
		shouldRemove := false

		// Check MaxAge policy
		if dfw.maxAge > 0 {
			age := now.Sub(file.modTime)
			if age > time.Duration(dfw.maxAge)*24*time.Hour {
				shouldRemove = true
			}
		}

		// Check MaxBackups policy (keep only N most recent files)
		if dfw.maxBackups > 0 && i >= dfw.maxBackups {
			shouldRemove = true
		}

		if shouldRemove {
			filesToRemove = append(filesToRemove, file.path)
		}
	}

	// Remove files
	for _, file := range filesToRemove {
		if err := os.Remove(file); err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: failed to remove old log file %s: %v\n", file, err)
		}
	}

	return nil
}

// Cleanup manually triggers cleanup of old log files
func (dfw *DailyFileWriter) Cleanup() error {
	dfw.mu.Lock()
	defer dfw.mu.Unlock()
	return dfw.cleanup()
}

// CompressFile manually compresses a specific log file
func (dfw *DailyFileWriter) CompressFile(filename string) error {
	return dfw.compressFile(filename)
}

// GetLogFiles returns a list of all log files in the directory
func (dfw *DailyFileWriter) GetLogFiles() ([]string, error) {
	pattern := filepath.Join(dfw.directory, "app-*.log*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}
	
	// Sort files by name (which includes date)
	sort.Strings(files)
	return files, nil
}

// Ensure DailyFileWriter implements io.Writer and io.Closer
var _ io.Writer = (*DailyFileWriter)(nil)
var _ io.Closer = (*DailyFileWriter)(nil)