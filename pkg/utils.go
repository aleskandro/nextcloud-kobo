package pkg

import (
	"fmt"
	"os"
	"path"
	"time"
)

func shouldDownloadFile(localFilePath string, remoteModTime time.Time, size int64) bool {
	info, err := os.Stat(localFilePath)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		fmt.Println("Error getting local file info:", err)
		return true
	}
	if info.Size() != size {
		return true
	}
	return remoteModTime.After(info.ModTime())
}

func removeRemotelyDeletedFiles(remoteFileMap map[string]string, localPath string) (err error) {
	files, _ := os.ReadDir(localPath)
	for _, file := range files {
		localFilePath := path.Join(localPath, file.Name())
		if _, ok := remoteFileMap[file.Name()]; !ok {
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
