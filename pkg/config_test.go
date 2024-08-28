package pkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary configuration file for testing
	configContent := `
remotes:
  - url: "http://example.com/s/xyz123"
    local_path: "sync_folder"
`
	tmpFile, err := os.CreateTemp("", "config*.yaml")
	assert.NoError(t, err)
	//nolint:errcheck
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte(configContent))
	assert.NoError(t, err)
	//nolint:errcheck
	tmpFile.Close()

	// Define base path for test
	basePath := "/base/path"

	// Test loading valid configuration
	config, err := LoadConfig(tmpFile.Name(), basePath)
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, 1, len(config.Remotes))
	assert.Equal(t, "http://example.com/s/xyz123", config.Remotes[0].URL)
	assert.Equal(t, filepath.Join(basePath, "sync_folder"), config.Remotes[0].LocalPath)
	assert.Equal(t, "/", config.Remotes[0].RemoteFolder)
	assert.Equal(t, "xyz123", config.Remotes[0].Username)
	assert.Equal(t, "example.com:sync_folder", config.Remotes[0].String())

	// Test loading configuration with a non-existent file
	config, err = LoadConfig("nonexistent.yaml", basePath)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "error reading config file")

	// Test loading configuration with invalid YAML content
	invalidConfigContent := `
remotes:
  - url: "http://example.com/s/xyz123"
    local_path: "sync_folder"
invalid_yaml
`
	tmpFile2, err := os.CreateTemp("", "config_invalid*.yaml")
	assert.NoError(t, err)
	//nolint:errcheck
	defer os.Remove(tmpFile2.Name())

	_, err = tmpFile2.Write([]byte(invalidConfigContent))
	assert.NoError(t, err)
	//nolint:errcheck
	tmpFile2.Close()

	config, err = LoadConfig(tmpFile2.Name(), basePath)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "error parsing config file")
}

func TestRemote_validateAndSetup(t *testing.T) {
	basePath := "/base/path"

	// Test case with a valid shared link
	remote := &Remote{
		URL:       "http://example.com/s/xyz123",
		LocalPath: "sync_folder",
	}

	err := remote.validateAndSetup(basePath)
	assert.NoError(t, err)
	assert.Equal(t, "/", remote.RemoteFolder)
	assert.Equal(t, "xyz123", remote.Username)
	assert.Equal(t, filepath.Join(basePath, "sync_folder"), remote.LocalPath)
	assert.Equal(t, "example.com:sync_folder", remote.String())

	// Test case with missing URL
	remote = &Remote{
		LocalPath: "sync_folder",
	}

	err = remote.validateAndSetup(basePath)
	assert.Error(t, err)
	assert.Equal(t, "URL is required", err.Error())

	// Test case with missing local path
	remote = &Remote{
		URL: "http://example.com/s/xyz123",
	}

	err = remote.validateAndSetup(basePath)
	assert.Error(t, err)
	assert.Equal(t, "local path is required", err.Error())

	// Test case with username set for a shared link
	remote = &Remote{
		URL:       "http://example.com/s/xyz123",
		Username:  "user",
		LocalPath: "sync_folder",
	}

	err = remote.validateAndSetup(basePath)
	assert.Error(t, err)
	assert.Equal(t, "username should not be set for shared links", err.Error())

	// Test case with remote folder set for a shared link
	remote = &Remote{
		URL:          "http://example.com/s/xyz123",
		RemoteFolder: "should_not_set",
		LocalPath:    "sync_folder",
	}

	err = remote.validateAndSetup(basePath)
	assert.Error(t, err)
	assert.Equal(t, "remote folder should not be set for shared links", err.Error())

	// Test case with invalid URL
	remote = &Remote{
		URL:       "://invalid-url",
		LocalPath: "sync_folder",
	}

	err = remote.validateAndSetup(basePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URL")
}
