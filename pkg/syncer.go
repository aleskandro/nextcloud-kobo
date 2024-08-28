package pkg

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/studio-b12/gowebdav"
)

type Syncer struct {
	config *Config
}

func NewSyncer(config *Config) *Syncer {
	return &Syncer{config: config}
}

func (c *Syncer) RunSync() (updatedFiles map[string][]string, err error) {
	updatedFiles = make(map[string][]string)
	log.Println("Running sync")
	for _, r := range c.config.Remotes {
		log.Println("Syncing remote", r.String())
		client := gowebdav.NewClient(r.remoteURL.String(), r.Username, r.Password)
		updatedFiles[r.String()], err = c.syncFolder(client, r.RemoteFolder, r.LocalPath)
		if err != nil {
			log.Println("error syncing folder", r.String(), err)
			return updatedFiles, fmt.Errorf("error syncing folder %s: %s", r.String(), err)
		}
		log.Println("Synced remote", r.String())
	}
	return
}

func (c *Syncer) syncFolder(client *gowebdav.Client, remotePath, localPath string) (updatedFiles []string, err error) {
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
		remoteFilePath := path.Join(remotePath, file.Name())
		localFilePath := path.Join(localPath, file.Name())
		log.Println("Checking file", remoteFilePath, localFilePath)
		if file.IsDir() {
			if ensureDirExists(localFilePath) != nil {
				return
			}
			var updatedFilesRec []string
			updatedFilesRec, err = c.syncFolder(client, remoteFilePath+"/", localFilePath)
			updatedFiles = append(updatedFiles, updatedFilesRec...)
			if err != nil {
				return
			}
		} else {
			localFileMap[localFilePath] = localFilePath
			if shouldDownloadFile(localFilePath, file.ModTime(), file.Size()) {
				log.Printf("Downloading file %s to %s\n", remoteFilePath, localFilePath)
				var data []byte
				data, err = client.Read(remoteFilePath)
				if err != nil {
					return
				}
				//#nosec G306
				err = os.WriteFile(localFilePath, data, 0644)
				if err != nil {
					return
				}
				updatedFiles = append(updatedFiles, localFilePath)
				log.Println("Downloaded file", localFilePath)
			} else {
				log.Println("Skipping file", remoteFilePath)
			}
		}
	}
	err = removeRemotelyDeletedFiles(localFileMap, localPath)
	return updatedFiles, err
}
