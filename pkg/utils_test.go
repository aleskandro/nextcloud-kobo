package pkg

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

// Helper function to create a temporary file for testing
func createTempFile(t *testing.T, content string, modTime time.Time) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	//nolint:errcheck
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	// Set the modification time
	if err := os.Chtimes(tmpFile.Name(), modTime, modTime); err != nil {
		t.Fatalf("Failed to set modification time: %v", err)
	}

	return tmpFile.Name()
}

func TestShouldDownloadFile(t *testing.T) {
	now := time.Now()
	oldTime := now.Add(-time.Hour)
	newTime := now.Add(time.Hour)

	tests := []struct {
		name           string
		localFilePath  string
		remoteModTime  time.Time
		size           int64
		expectedResult bool
	}{
		{
			name:           "File does not exist",
			localFilePath:  "nonexistent.txt",
			remoteModTime:  now,
			size:           100,
			expectedResult: true,
		},
		{
			name:           "File exists with different size",
			localFilePath:  createTempFile(t, "content", oldTime),
			remoteModTime:  now,
			size:           200, // Different size
			expectedResult: true,
		},
		{
			name:           "File exists with older mod time",
			localFilePath:  createTempFile(t, "content", oldTime),
			remoteModTime:  newTime,
			size:           7, // Same size as content
			expectedResult: true,
		},
		{
			name:           "File exists with same mod time and size",
			localFilePath:  createTempFile(t, "content", now),
			remoteModTime:  now,
			size:           7, // Same size as content
			expectedResult: false,
		},
	}

	for _, tt := range tests[1:] {
		//nolint:errcheck
		defer os.Remove(tt.localFilePath)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldDownloadFile(tt.localFilePath, tt.remoteModTime, tt.size)
			if result != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestRemoveRemotelyDeletedFiles(t *testing.T) {
	// Create a temporary directory for testing
	localDir, err := os.MkdirTemp("", "localdir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	//nolint:errcheck
	defer os.RemoveAll(localDir) // Clean up after the test

	// Create some local files
	localFile1 := filepath.Join(localDir, "file1.txt")
	localFile2 := filepath.Join(localDir, "file2.txt")
	if err := os.WriteFile(localFile1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(localFile2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Define remote files map (file2 is missing)
	remoteFiles := map[string]string{
		path.Join(localDir, "file1.txt"): "",
	}

	// Run the function to test
	err = removeRemotelyDeletedFiles(remoteFiles, localDir)
	if err != nil {
		t.Fatalf("Function returned an error: %v", err)
	}

	// Check if the correct files were removed
	if _, err := os.Stat(localFile1); os.IsNotExist(err) {
		t.Errorf("file1.txt should not be deleted")
	}
	if _, err := os.Stat(localFile2); err == nil {
		t.Errorf("file2.txt should have been deleted")
	}
}
