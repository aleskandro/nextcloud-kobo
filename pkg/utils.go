package pkg

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"
)

// 8 MB limit per file to mitigate G110: (CWE-409): Potential DoS vulnerability via decompression bomb
const maxFileSize = int64(16 * 1024 * 1024)

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

func extractTarGz(gzipStream io.ReadCloser) error {
	// Create a gzip reader from the io.ReadCloser
	gzipReader, err := gzip.NewReader(gzipStream)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	//nolint:errcheck
	defer gzipReader.Close()

	// Create a tar reader from the decompressed gzip reader
	tarReader := tar.NewReader(gzipReader)

	// Iterate through the files in the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// End of archive
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Handle different file types (e.g., regular files, directories)
		switch header.Typeflag {
		case tar.TypeDir:
			// Create the directory if it doesn't exist
			if err := os.MkdirAll(path.Join("/", filepath.Clean(header.Name)), os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", header.Name, err)
			}

		case tar.TypeReg:
			// Create the file
			outFile, err := os.Create(path.Join("/", header.Name)) // #nosec G305
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", header.Name, err)
			}
			//nolint:errcheck
			defer outFile.Close()

			// Copy file content from the tar archive
			limitReader := io.LimitReader(tarReader, maxFileSize)
			if _, err := io.Copy(outFile, limitReader); err != nil {
				return fmt.Errorf("failed to write file %s: %w", header.Name, err)
			}

		default:
			// Handle other file types if needed
			fmt.Printf("Unknown file type: %c in %s\n", header.Typeflag, header.Name)
		}
	}

	return nil
}
