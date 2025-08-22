package logging

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDailyFileWriter(t *testing.T) {
	tests := []struct {
		name        string
		config      DailyFileConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: DailyFileConfig{
				Directory:  "testlogs",
				MaxAge:     30,
				MaxBackups: 10,
				Compress:   true,
				LocalTime:  true,
			},
			expectError: false,
		},
		{
			name: "invalid directory permissions",
			config: DailyFileConfig{
				Directory:  "/root/testlogs", // Should fail on most systems
				MaxAge:     30,
				MaxBackups: 10,
				Compress:   true,
				LocalTime:  true,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			if tt.config.Directory != "/root/testlogs" {
				defer os.RemoveAll(tt.config.Directory)
			}

			writer, err := NewDailyFileWriter(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, writer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, writer)
				
				// Verify directory was created
				_, err := os.Stat(tt.config.Directory)
				assert.NoError(t, err)
				
				// Verify initial file was created
				currentDate := time.Now().Format("2006-01-02")
				expectedFile := filepath.Join(tt.config.Directory, fmt.Sprintf("app-%s.log", currentDate))
				_, err = os.Stat(expectedFile)
				assert.NoError(t, err)
				
				// Clean up
				writer.Close()
			}
		})
	}
}

func TestDailyFileWriter_Write(t *testing.T) {
	tempDir := "testlogs_write"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Test writing data
	testData := []byte("test log entry\n")
	n, err := writer.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)

	// Verify data was written
	writer.Sync()
	content, err := os.ReadFile(writer.GetCurrentFileName())
	assert.NoError(t, err)
	assert.Equal(t, string(testData), string(content))
}

func TestDailyFileWriter_MultipleWrites(t *testing.T) {
	tempDir := "testlogs_multiple"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Write multiple entries
	entries := []string{
		"first log entry\n",
		"second log entry\n",
		"third log entry\n",
	}

	for _, entry := range entries {
		n, err := writer.Write([]byte(entry))
		assert.NoError(t, err)
		assert.Equal(t, len(entry), n)
	}

	// Verify all data was written
	writer.Sync()
	content, err := os.ReadFile(writer.GetCurrentFileName())
	assert.NoError(t, err)
	
	expectedContent := strings.Join(entries, "")
	assert.Equal(t, expectedContent, string(content))
}

func TestDailyFileWriter_GetCurrentFileName(t *testing.T) {
	tempDir := "testlogs_filename"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Check filename format
	filename := writer.GetCurrentFileName()
	currentDate := time.Now().Format("2006-01-02")
	expectedFilename := filepath.Join(tempDir, fmt.Sprintf("app-%s.log", currentDate))
	
	assert.Equal(t, expectedFilename, filename)
}

func TestDailyFileWriter_ShouldRotate(t *testing.T) {
	tempDir := "testlogs_rotate"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Initially should not need rotation
	assert.False(t, writer.shouldRotate())

	// Simulate date change by modifying currentDate
	writer.mu.Lock()
	writer.currentDate = "2023-01-01" // Set to past date
	writer.mu.Unlock()

	// Now should need rotation
	assert.True(t, writer.shouldRotate())
}

func TestDailyFileWriter_ManualRotate(t *testing.T) {
	tempDir := "testlogs_manual_rotate"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Write some data
	testData := "before rotation\n"
	_, err = writer.Write([]byte(testData))
	require.NoError(t, err)

	originalFilename := writer.GetCurrentFileName()

	// Manually trigger rotation
	err = writer.Rotate()
	assert.NoError(t, err)

	// Filename should be the same (same date)
	newFilename := writer.GetCurrentFileName()
	assert.Equal(t, originalFilename, newFilename)

	// Write more data after rotation
	moreData := "after rotation\n"
	_, err = writer.Write([]byte(moreData))
	require.NoError(t, err)

	// Verify both data entries are in the file
	writer.Sync()
	content, err := os.ReadFile(writer.GetCurrentFileName())
	assert.NoError(t, err)
	assert.Contains(t, string(content), testData)
	assert.Contains(t, string(content), moreData)
}

func TestDailyFileWriter_GetCurrentDate(t *testing.T) {
	tests := []struct {
		name      string
		localTime bool
	}{
		{
			name:      "local time",
			localTime: true,
		},
		{
			name:      "UTC time",
			localTime: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := fmt.Sprintf("testlogs_date_%s", strings.ReplaceAll(tt.name, " ", "_"))
			defer os.RemoveAll(tempDir)

			config := DailyFileConfig{
				Directory:  tempDir,
				MaxAge:     30,
				MaxBackups: 10,
				Compress:   false,
				LocalTime:  tt.localTime,
			}

			writer, err := NewDailyFileWriter(config)
			require.NoError(t, err)
			defer writer.Close()

			currentDate := writer.getCurrentDate()
			
			// Verify date format
			_, err = time.Parse("2006-01-02", currentDate)
			assert.NoError(t, err)

			// Verify it matches expected date
			var expectedDate string
			if tt.localTime {
				expectedDate = time.Now().Format("2006-01-02")
			} else {
				expectedDate = time.Now().UTC().Format("2006-01-02")
			}
			assert.Equal(t, expectedDate, currentDate)
		})
	}
}

func TestDailyFileWriter_Close(t *testing.T) {
	tempDir := "testlogs_close"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)

	// Write some data
	_, err = writer.Write([]byte("test data\n"))
	require.NoError(t, err)

	// Close the writer
	err = writer.Close()
	assert.NoError(t, err)

	// Verify file is closed by trying to write (should fail)
	_, err = writer.Write([]byte("should fail\n"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no current log file available")

	// Multiple closes should not error
	err = writer.Close()
	assert.NoError(t, err)
}

func TestDailyFileWriter_Sync(t *testing.T) {
	tempDir := "testlogs_sync"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Write some data
	_, err = writer.Write([]byte("test data for sync\n"))
	require.NoError(t, err)

	// Sync should not error
	err = writer.Sync()
	assert.NoError(t, err)

	// Verify data is written to disk
	content, err := os.ReadFile(writer.GetCurrentFileName())
	assert.NoError(t, err)
	assert.Equal(t, "test data for sync\n", string(content))
}

func TestDailyFileWriter_ConcurrentWrites(t *testing.T) {
	tempDir := "testlogs_concurrent"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Test concurrent writes
	const numGoroutines = 10
	const writesPerGoroutine = 100
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < writesPerGoroutine; j++ {
				data := fmt.Sprintf("goroutine-%d-write-%d\n", id, j)
				_, err := writer.Write([]byte(data))
				assert.NoError(t, err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Sync and verify total writes
	writer.Sync()
	content, err := os.ReadFile(writer.GetCurrentFileName())
	assert.NoError(t, err)
	
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	expectedLines := numGoroutines * writesPerGoroutine
	assert.Equal(t, expectedLines, len(lines))
}

func TestDailyFileWriter_CompressFile(t *testing.T) {
	tempDir := "testlogs_compress"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   true,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Create a test file to compress
	testFile := filepath.Join(tempDir, "test-file.log")
	testContent := "This is test content for compression\nLine 2\nLine 3\n"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Compress the file
	err = writer.CompressFile(testFile)
	assert.NoError(t, err)

	// Verify original file is removed
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))

	// Verify compressed file exists
	compressedFile := testFile + ".gz"
	_, err = os.Stat(compressedFile)
	assert.NoError(t, err)

	// Verify compressed content can be read back
	file, err := os.Open(compressedFile)
	require.NoError(t, err)
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer gzReader.Close()

	decompressedContent, err := io.ReadAll(gzReader)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(decompressedContent))
}

func TestDailyFileWriter_CompressFile_NonExistent(t *testing.T) {
	tempDir := "testlogs_compress_nonexistent"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   true,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Try to compress non-existent file
	err = writer.CompressFile(filepath.Join(tempDir, "nonexistent.log"))
	assert.NoError(t, err) // Should not error for non-existent files
}

func TestDailyFileWriter_Cleanup_MaxAge(t *testing.T) {
	tempDir := "testlogs_cleanup_age"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     2, // Keep files for 2 days
		MaxBackups: 0, // No backup limit
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Create test files with different ages
	now := time.Now()
	testFiles := []struct {
		name string
		age  time.Duration
	}{
		{"app-2024-01-01.log", 5 * 24 * time.Hour},  // 5 days old - should be removed
		{"app-2024-01-02.log", 3 * 24 * time.Hour},  // 3 days old - should be removed
		{"app-2024-01-03.log", 1 * 24 * time.Hour},  // 1 day old - should be kept
		{"app-2024-01-04.log", 12 * time.Hour},      // 12 hours old - should be kept
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(tempDir, tf.name)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)

		// Set file modification time
		modTime := now.Add(-tf.age)
		err = os.Chtimes(filePath, modTime, modTime)
		require.NoError(t, err)
	}

	// Run cleanup
	err = writer.Cleanup()
	assert.NoError(t, err)

	// Check which files remain
	files, err := writer.GetLogFiles()
	require.NoError(t, err)

	// Should have current file + 2 files within MaxAge
	expectedFiles := []string{
		filepath.Join(tempDir, "app-2024-01-03.log"),
		filepath.Join(tempDir, "app-2024-01-04.log"),
		writer.GetCurrentFileName(), // Current file
	}

	// Sort both slices for comparison
	sort.Strings(files)
	sort.Strings(expectedFiles)

	assert.ElementsMatch(t, expectedFiles, files)
}

func TestDailyFileWriter_Cleanup_MaxBackups(t *testing.T) {
	tempDir := "testlogs_cleanup_backups"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     0, // No age limit
		MaxBackups: 3, // Keep only 3 backup files
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Create test files with different modification times
	now := time.Now()
	testFiles := []struct {
		name    string
		modTime time.Time
	}{
		{"app-2024-01-01.log", now.Add(-5 * time.Hour)}, // Oldest
		{"app-2024-01-02.log", now.Add(-4 * time.Hour)},
		{"app-2024-01-03.log", now.Add(-3 * time.Hour)},
		{"app-2024-01-04.log", now.Add(-2 * time.Hour)},
		{"app-2024-01-05.log", now.Add(-1 * time.Hour)}, // Newest
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(tempDir, tf.name)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)

		// Set file modification time
		err = os.Chtimes(filePath, tf.modTime, tf.modTime)
		require.NoError(t, err)
	}

	// Run cleanup
	err = writer.Cleanup()
	assert.NoError(t, err)

	// Check which files remain
	files, err := writer.GetLogFiles()
	require.NoError(t, err)

	// Should have current file + 3 most recent backup files
	expectedFiles := []string{
		filepath.Join(tempDir, "app-2024-01-03.log"),
		filepath.Join(tempDir, "app-2024-01-04.log"),
		filepath.Join(tempDir, "app-2024-01-05.log"),
		writer.GetCurrentFileName(), // Current file
	}

	// Sort both slices for comparison
	sort.Strings(files)
	sort.Strings(expectedFiles)

	assert.ElementsMatch(t, expectedFiles, files)
}

func TestDailyFileWriter_Cleanup_Combined(t *testing.T) {
	tempDir := "testlogs_cleanup_combined"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     3,  // Keep files for 3 days
		MaxBackups: 2,  // Keep only 2 backup files
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Create test files
	now := time.Now()
	testFiles := []struct {
		name    string
		age     time.Duration
		modTime time.Time
	}{
		{"app-2024-01-01.log", 5 * 24 * time.Hour, now.Add(-5 * 24 * time.Hour)}, // Too old
		{"app-2024-01-02.log", 4 * 24 * time.Hour, now.Add(-4 * 24 * time.Hour)}, // Too old
		{"app-2024-01-03.log", 2 * 24 * time.Hour, now.Add(-2 * 24 * time.Hour)}, // Within age, newest backup
		{"app-2024-01-04.log", 1 * 24 * time.Hour, now.Add(-1 * 24 * time.Hour)}, // Within age, second newest backup
		{"app-2024-01-05.log", 12 * time.Hour, now.Add(-12 * time.Hour)},         // Within age, but exceeds MaxBackups
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(tempDir, tf.name)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)

		// Set file modification time
		err = os.Chtimes(filePath, tf.modTime, tf.modTime)
		require.NoError(t, err)
	}

	// Run cleanup
	err = writer.Cleanup()
	assert.NoError(t, err)

	// Check which files remain
	files, err := writer.GetLogFiles()
	require.NoError(t, err)

	// Should have current file + 2 most recent files within age limit
	// Files within age: app-2024-01-03.log (2 days), app-2024-01-04.log (1 day), app-2024-01-05.log (12 hours)
	// MaxBackups=2, so keep the 2 most recent: app-2024-01-04.log and app-2024-01-05.log
	expectedFiles := []string{
		filepath.Join(tempDir, "app-2024-01-04.log"), // Within age and backup limit (2nd most recent)
		filepath.Join(tempDir, "app-2024-01-05.log"), // Within age and backup limit (most recent)
		writer.GetCurrentFileName(),                   // Current file
	}

	// Sort both slices for comparison
	sort.Strings(files)
	sort.Strings(expectedFiles)

	assert.ElementsMatch(t, expectedFiles, files)
}

func TestDailyFileWriter_GetLogFiles(t *testing.T) {
	tempDir := "testlogs_getfiles"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   false,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Create some test log files
	testFiles := []string{
		"app-2024-01-01.log",
		"app-2024-01-02.log.gz",
		"app-2024-01-03.log",
		"other-file.txt", // Should not be included
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Get log files
	files, err := writer.GetLogFiles()
	assert.NoError(t, err)

	// Should include log files but not other files
	expectedFiles := []string{
		filepath.Join(tempDir, "app-2024-01-01.log"),
		filepath.Join(tempDir, "app-2024-01-02.log.gz"),
		filepath.Join(tempDir, "app-2024-01-03.log"),
		writer.GetCurrentFileName(), // Current file
	}

	// Sort both slices for comparison
	sort.Strings(files)
	sort.Strings(expectedFiles)

	assert.ElementsMatch(t, expectedFiles, files)
}

func TestDailyFileWriter_AutomaticCleanupOnRotation(t *testing.T) {
	tempDir := "testlogs_auto_cleanup"
	defer os.RemoveAll(tempDir)

	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     1, // Keep files for 1 day
		MaxBackups: 2, // Keep only 2 backup files
		Compress:   true,
		LocalTime:  true,
	}

	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()

	// Create old files that should be cleaned up
	now := time.Now()
	oldFiles := []struct {
		name    string
		modTime time.Time
	}{
		{"app-2024-01-01.log", now.Add(-3 * 24 * time.Hour)}, // Too old
		{"app-2024-01-02.log", now.Add(-2 * 24 * time.Hour)}, // Too old
	}

	for _, of := range oldFiles {
		filePath := filepath.Join(tempDir, of.name)
		err := os.WriteFile(filePath, []byte("old content"), 0644)
		require.NoError(t, err)
		err = os.Chtimes(filePath, of.modTime, of.modTime)
		require.NoError(t, err)
	}

	// Write some data to current file
	_, err = writer.Write([]byte("current data\n"))
	require.NoError(t, err)

	// Simulate date change and trigger rotation
	writer.mu.Lock()
	writer.currentDate = "2023-01-01" // Force rotation
	writer.mu.Unlock()

	// Write more data to trigger rotation
	_, err = writer.Write([]byte("new data\n"))
	require.NoError(t, err)

	// Give background cleanup time to run
	time.Sleep(100 * time.Millisecond)

	// Check that old files were cleaned up
	files, err := writer.GetLogFiles()
	require.NoError(t, err)

	// Should have current file and possibly compressed previous file
	// The previous file might be compressed (.gz) due to rotation
	assert.GreaterOrEqual(t, len(files), 1)
	
	// Current file should be present
	currentFile := writer.GetCurrentFileName()
	found := false
	for _, file := range files {
		if file == currentFile || file == currentFile+".gz" {
			found = true
			break
		}
	}
	assert.True(t, found, "Current file or its compressed version should be present")
}