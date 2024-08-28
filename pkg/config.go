package pkg

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Remotes    []Remote `yaml:"remotes"`
	AutoUpdate bool     `yaml:"auto_update,omitempty"`
	basePath   string   `yaml:"-"`
}

type Remote struct {
	// URL can be either a full URL to a Nextcloud shared link or the NextCloud host URL
	// If URL is a shared link, the username is not required, and we will extract it from the URL
	URL string `yaml:"url"`
	// Username is the username to use for authentication. It is only required if URL is not a shared link, and you
	// want to authenticate with a specific username to sync a private folder
	Username string `yaml:"username,omitempty"`
	// Password is the password to use for authentication. It is only required if URL is a protected shared link, or
	// you want to authenticate as a specific user to sync a private folder.
	Password string `yaml:"password,omitempty"`
	// RemoteFolder is the folder on the remote server to sync.
	// If not specified, the root folder will be synced by default.
	// When syncing a shared link, this should not be set.
	RemoteFolder string `yaml:"remote_folder,omitempty"`
	// LocalPath is the local path to sync the remote folder to
	LocalPath string `yaml:"local_path"`

	// remoteURL is the parsed and processed URL that we will use to connect to the remote server
	remoteURL    *url.URL
	printableURL string
}

func LoadConfig(configFilePath, basePath string) (*Config, error) {
	config := &Config{
		basePath: basePath,
	}
	configFilePath = filepath.Clean(configFilePath)
	configFile, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	err = yaml.Unmarshal(configFile, config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}
	for i := range config.Remotes {
		err = config.Remotes[i].validateAndSetup(filepath.Clean(basePath))
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}

func (r *Remote) validateAndSetup(basePath string) error {
	if r.URL == "" {
		return fmt.Errorf("URL is required")
	}
	if r.LocalPath == "" {
		return fmt.Errorf("local path is required")
	}
	// if URL is a shared link, username should not be set
	if r.Username != "" && strings.Contains(r.URL, "/s/") {
		return fmt.Errorf("username should not be set for shared links")
	}

	// if URL is a shared link, the remote folder should not be set
	if r.RemoteFolder != "" && strings.Contains(r.URL, "/s/") {
		return fmt.Errorf("remote folder should not be set for shared links")
	}

	// We set the remote folder to the root folder if it is not set (or it is a shared link)
	if r.RemoteFolder == "" {
		r.RemoteFolder = "/"
	}

	baseURL, err := url.Parse(r.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %s", r.URL)
	}

	// Check if URL is a shared link (contains "/s/")
	if strings.Contains(baseURL.Path, "/s/") {
		parts := strings.Split(baseURL.Path, "/s/")
		if len(parts) < 2 || parts[1] == "" {
			return fmt.Errorf("invalid URL: %s", r.URL)
		}
		// Extract the username (share ID) from the URL
		r.Username = parts[1]
		// Remove "/index.php" if it exists in the path
		baseURL.Path = strings.Replace(parts[0], "/index.php", "", -1)
	}
	// Resolve "public.php/webdav" relative to the base URL
	webdavPath, _ := url.Parse("public.php/webdav")
	r.remoteURL = baseURL.ResolveReference(webdavPath)
	r.printableURL = fmt.Sprintf("%s:%s", r.remoteURL.Host, r.LocalPath)
	r.LocalPath = path.Join(basePath, r.LocalPath)
	return nil
}

func (r *Remote) String() string {
	return r.printableURL
}
