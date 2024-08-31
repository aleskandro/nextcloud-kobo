package pkg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/google/go-github/v55/github"

	"github.com/studio-b12/gowebdav"
)

func (n *NetworkConnectionReconciler) sync(ctx context.Context) {
	var (
		filesMap      map[string][]string
		nUpdatedFiles int
		err           error
	)
	checkNetworkCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if err = checkNetwork(checkNetworkCtx); err != nil {
		log.Println("Network connection failed", err)
		if !errors.Is(err, networkConnectionFailedErr) {
			n.toastsChan <- fmt.Sprintf("Failed to sync: %s\n%s", err.Error(), generateFilesString(filesMap))
		}
		return
	}
	n.toastsChan <- "Syncing with Nextcloud..."
	filesMap, err = n.syncRemotes(ctx)
	if err != nil {
		log.Println("An error occurred during synchronization", err)
	}
	for _, files := range filesMap {
		nUpdatedFiles += len(files)
	}
	if nUpdatedFiles > 0 {
		n.toastsChan <- fmt.Sprintf("Synced %d files:\n%s", nUpdatedFiles, generateFilesString(filesMap))
	} else {
		n.toastsChan <- "No files updated"
		log.Println("No files updated")
	}
	log.Println("Sync successful")
	n.rescanBooks()
}

func (n *NetworkConnectionReconciler) syncRemotes(ctx context.Context) (updatedFiles map[string][]string, err error) {
	updatedFiles = make(map[string][]string)
	log.Println("Running sync")
	for _, r := range n.config.Remotes {
		if err = ctx.Err(); err != nil {
			log.Println("The context has been canceled. Interrupting...")
			err = ctx.Err()
			return
		}
		client := gowebdav.NewClient(r.remoteURL.String(), r.Username, r.Password)
		// 10 Mb/s * 4 min * 60 s/min * 1/8 B/b = 300 MB per file/book max with a 10 Mbps connection(?)
		client.SetTimeout(time.Minute * 4)
		updatedFiles[r.String()], err = n.syncFolder(client, ctx, r.RemoteFolder, r.LocalPath)
		if err != nil {
			log.Println("error syncing folder", r.String(), err)
			return updatedFiles, fmt.Errorf("error syncing folder %s: %s", r.String(), err)
		}
		log.Println("Synced remote", r.String())
	}
	return
}

func (n *NetworkConnectionReconciler) syncFolder(client *gowebdav.Client, ctx context.Context, remotePath, localPath string) (updatedFiles []string, err error) {
	var remoteFiles []os.FileInfo
	updatedFiles = []string{}
	remoteFiles, err = client.ReadDir(remotePath)
	if err != nil {
		return
	}

	if ensureDirExists(localPath) != nil {
		return
	}
	localFileMap := make(map[string]string)
	for _, file := range remoteFiles {
		if ctx.Err() != nil {
			log.Println("The context has been canceled. Interrupting...")
			err = ctx.Err()
			return
		}
		remoteFilePath := path.Join(remotePath, file.Name())
		localFilePath := path.Join(localPath, file.Name())
		log.Println("Checking file", remoteFilePath, localFilePath)
		if file.IsDir() {
			log.Println(remoteFilePath, "is a dir. Executing recursion...", localFilePath)
			if ensureDirExists(localFilePath) != nil {
				return
			}
			var updatedFilesRec []string
			updatedFilesRec, err = n.syncFolder(client, ctx, remoteFilePath+"/", localFilePath)
			updatedFiles = append(updatedFiles, updatedFilesRec...)
			if err != nil {
				return
			}
		} else {
			localFileMap[localFilePath] = localFilePath
			if shouldDownloadFile(localFilePath, file.ModTime(), file.Size()) {
				log.Printf("Downloading file %s to %s\n", remoteFilePath, localFilePath)
				if err = downloadFile(client, remoteFilePath, localFilePath); err != nil {
					return
				}
				updatedFiles = append(updatedFiles, localFilePath)
				log.Println("Downloaded file", localFilePath)
				n.toastsChan <- fmt.Sprintf("Downloaded %s", remoteFilePath)
			} else {
				log.Println("Skipping file", remoteFilePath)
			}
		}
	}
	err = removeRemotelyDeletedFiles(localFileMap, localPath)
	return updatedFiles, err
}

func (n *NetworkConnectionReconciler) updateNow() {
	// Check the latest version on GitHub
	cli := github.NewClient(nil)
	release, _, err := cli.Repositories.GetLatestRelease(context.Background(),
		n.config.RepoOwner, n.config.RepoName)
	// If we can't get the latest release, don't update
	if err != nil {
		log.Println("Failed to get latest release", err)
		return
	}
	// get the latest updated version stored in the config
	version, err := os.ReadFile(path.Join(n.config.configPath, "version.txt"))
	if err != nil && !os.IsNotExist(err) {
		log.Println("Failed to read version file", err)
		return
	}
	if os.IsNotExist(err) {
		log.Println("Version file not found, updating to latest release")
	} else {
		log.Println("Current version:", string(version), "Latest version:", *release.TagName)
	}
	if string(version) == *release.TagName {
		log.Println("Already up to date")
		return
	}
	// Download the latest release
	asset := *release.Assets[0]
	resp, err := http.Get(*asset.BrowserDownloadURL)
	if err != nil {
		log.Println("Failed to download latest release", err)
		return
	}
	//nolint:errcheck
	defer resp.Body.Close()

	// Save the latest release to a file
	file, err := os.Create(path.Join(n.config.configPath, "nextcloud-kobo.tar.gz"))
	if err != nil {
		log.Println("Failed to create release file", err)
		return
	}
	//nolint:errcheck
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Println("Failed to write release file:", err)
		return
	}
	// Write the latest release to a file
	versionFile, err := os.Create(path.Join(n.config.configPath, "version.txt"))
	if err != nil {
		log.Println("Failed to create version file", err)
		return
	}
	_, err = versionFile.Write([]byte(*release.TagName))
	if err != nil {
		log.Println("Failed to write version file", err)
		return
	}
	log.Println("Auto update successful")
	n.toastsChan <- "An update for Nextcloud-Kobo is available"
	os.Exit(0) // Exit to restart the application
}

func checkNetwork(ctx context.Context) error {
	// Wait for the network to be fully connected
	for i := 0; i < 10; i++ {
		// Check if a web request to google is successful
		client := &http.Client{
			Timeout: 5 * time.Second,
		}
		req, err := http.NewRequestWithContext(ctx, "GET", "http://www.google.com", nil)
		if err != nil {
			log.Println("Fatal error", err)
			return err
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			log.Printf("HTTP request #%d/10 successful\n", i+1)
			//nolint:errcheck
			resp.Body.Close()
			return nil
		}
		log.Printf("HTTP request #%d/10 failed: %v\n", i+1, err)
		time.Sleep(time.Second)
	}
	return fmt.Errorf("network connection failed")
}

func generateFilesString(filesMap map[string][]string) (filesString string) {
	for remote, files := range filesMap {
		filesString += fmt.Sprintf("Remote: %s\n", remote)
		for _, file := range files {
			filesString += fmt.Sprintf("  - %s\n", file)
		}
	}
	return
}

func downloadFile(client *gowebdav.Client, remoteFilePath, localFilePath string) error {
	log.Printf("Downloading file %s to %s\n", remoteFilePath, localFilePath)
	remoteFileReader, err := client.ReadStream(remoteFilePath) // Assuming ReadStream returns an io.ReadCloser
	if err != nil {
		return fmt.Errorf("error reading remote file %s: %w", remoteFilePath, err)
	}
	//nolint:errcheck
	defer remoteFileReader.Close()

	localFileWriter, err := os.Create(path.Clean(localFilePath))
	if err != nil {
		return fmt.Errorf("error creating local file %s: %w", localFilePath, err)
	}
	//nolint:errcheck
	defer localFileWriter.Close()

	if _, err := io.Copy(localFileWriter, remoteFileReader); err != nil {
		return fmt.Errorf("error writing to local file %s: %w", localFilePath, err)
	}

	return nil
}
