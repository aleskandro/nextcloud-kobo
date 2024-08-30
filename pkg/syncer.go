package pkg

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/studio-b12/gowebdav"
)

func (n *NetworkConnectionReconciler) runSync(ctx context.Context) (updatedFiles map[string][]string, err error) {
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
				n.keepNetworkAlive()
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
				n.notifyNickel(fmt.Sprintf("Downloaded %s", remoteFilePath))
			} else {
				log.Println("Skipping file", remoteFilePath)
			}
		}
	}
	err = removeRemotelyDeletedFiles(localFileMap, localPath)
	return updatedFiles, err
}
