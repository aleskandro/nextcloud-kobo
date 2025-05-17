package pkg

import (
	"log"
	"os"
	"path"
	"strings"
	"time"
)

func shouldDownloadFile(localFilePath string, remoteModTime time.Time, size int64) bool {
	info, err := os.Stat(localFilePath)
	log.Println("Checking existence of ", localFilePath)
	return err != nil || info.Size() != size || remoteModTime.After(info.ModTime())
}

func removeRemotelyDeletedFiles(localFileMap map[string]string, localPath string) (err error) {
	files, _ := os.ReadDir(localPath)
	for _, file := range files {
		localFilePath := path.Join(localPath, file.Name())
		if strings.HasSuffix(file.Name(), ".sdr") || strings.Contains(file.Name(), ".sdr/") {
			// Ignore .sdr annotations written by KOReader
			log.Println("Skipping file", file.Name(), localFilePath)
			continue
		}
		if _, ok := localFileMap[localFilePath]; !ok {
			log.Println("Removing file", file.Name(), localFilePath)
			err = os.RemoveAll(localFilePath)
			if err != nil {
				return
			}
		}
	}
	return
}

func ensureDirExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, os.ModePerm)
	}
	return nil
}
