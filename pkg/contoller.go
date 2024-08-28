package pkg

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/google/go-github/v55/github"
)

type NetworkConnectionReconciler struct {
	conn   *dbus.Conn
	ch     chan *dbus.Signal
	syncer *Syncer
}

var networkConnectionFailedErr = fmt.Errorf("network connection failed")

func NewNetworkConnectionReconciler(config *Config) *NetworkConnectionReconciler {
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Fatalf("Failed to connect to system bus: %v", err)
	}
	ch := make(chan *dbus.Signal, 10)
	call := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',interface='com.github.shermp.nickeldbus',member='wmNetworkConnected',path='/nickeldbus'")
	if call.Err != nil {
		log.Fatalf("Failed to add D-Bus match: %v", call.Err)
	}
	conn.Signal(ch)
	return &NetworkConnectionReconciler{
		conn:   conn,
		ch:     ch,
		syncer: NewSyncer(config),
	}
}

func (n *NetworkConnectionReconciler) Run(ctx context.Context) {
	defer func() {
		fmt.Println("Exiting network connection reconciler")
		n.conn.RemoveSignal(n.ch)
		close(n.ch)
		//nolint:errcheck
		n.conn.Close()
	}()
	for {
		fmt.Println("Listening for network connection signals from Nickel...")
		select {
		case <-ctx.Done():
			fmt.Println("Context done")
			return
		case signal, ok := <-n.ch:
			if !ok {
				log.Println("Signal channel closed")
				return
			}
			if signal == nil {
				log.Println("Received nil signal")
				continue
			}
			fmt.Printf("Received signal: %s\n", signal.Name)
			// Check if the signal is the one we are interested in
			if signal.Name != "com.github.shermp.nickeldbus.wmNetworkConnected" {
				log.Println("Received unexpected signal", signal.Name)
				continue
			}
			err := n.handleWmNetworkConnected()
			if err != nil {
				log.Println("Failed to handle network connected signal", err)
			}
		}
	}
}

func (n *NetworkConnectionReconciler) handleWmNetworkConnected() error {
	filesMap, nUpdatedFiles, err := n.SyncNow()
	if err != nil {
		log.Println("Failed to sync", err)
		if errors.Is(err, networkConnectionFailedErr) {
			return nil
		}
		n.notifyNickel(fmt.Sprintf("Failed to sync: %s\n%s", err.Error(), generateFilesString(filesMap)))
		return err
	}
	if nUpdatedFiles > 0 {
		n.notifyNickel(fmt.Sprintf("Synced %d files:\n%s", nUpdatedFiles, generateFilesString(filesMap)))
	}
	log.Println("Sync successful")
	if n.syncer.config.AutoUpdate && n.UpdateNow() {
		log.Println("Auto update successful")
		n.notifyNickel("Nextcloud-Kobo Syncer updated successfully")
	}
	return nil
}

func (n *NetworkConnectionReconciler) UpdateNow() bool {
	// Check the latest version on GitHub
	cli := github.NewClient(nil)
	release, _, err := cli.Repositories.GetLatestRelease(context.Background(), "aleskandro", "nextcloud-kobo")
	// If we can't get the latest release, don't update
	if err != nil {
		log.Println("Failed to get latest release", err)
		return false
	}
	// get the latest updated version stored in the config
	version, err := os.ReadFile(path.Join(n.syncer.config.basePath, "version.txt"))
	if err != nil && !os.IsNotExist(err) {
		log.Println("Failed to read version file", err)
		return false
	}
	if string(version) == *release.TagName {
		log.Println("Already up to date")
		return false
	}
	// Download the latest release
	asset := *release.Assets[0]
	resp, err := http.Get(*asset.BrowserDownloadURL)
	if err != nil {
		log.Println("Failed to download latest release", err)
		return false
	}
	//nolint:errcheck
	defer resp.Body.Close()

	// The latest release is a KoboRoot.tgz file. Let's extract it in the / directory
	// This will overwrite the existing files.
	err = extractTarGz(resp.Body)
	if err != nil {
		log.Println("Failed to extract tar.gz", err)
		return false
	}
	// Write the latest release to a file
	versionFile, err := os.Create(path.Join(n.syncer.config.basePath, "version.txt"))
	if err != nil {
		log.Println("Failed to create version file", err)
		return false
	}
	_, err = versionFile.Write([]byte(*release.TagName))
	if err != nil {
		log.Println("Failed to write version file", err)
		return false
	}
	return true
}

func (n *NetworkConnectionReconciler) SyncNow() (filesMap map[string][]string, nUpdatedFiles int, err error) {
	if err = checkNetwork(); err != nil {
		log.Println("Network connection failed", err)
		return filesMap, 0, networkConnectionFailedErr
	}

	filesMap, err = n.syncer.RunSync()
	if err != nil {
		return
	}
	for _, files := range filesMap {
		nUpdatedFiles += len(files)
	}
	if nUpdatedFiles == 0 {
		log.Println("No files updated")
		return
	}
	err = n.rescanBooks()
	if err != nil {
		return
	}
	return
}

func (n *NetworkConnectionReconciler) rescanBooks() error {
	obj := n.conn.Object("com.github.shermp.nickeldbus", "/nickeldbus")
	call := obj.Call("com.github.shermp.nickeldbus.pfmRescanBooks", 0)
	if call.Err != nil {
		log.Println("Failed to rescan books", call.Err)
		return call.Err
	}
	return nil
}

func (n *NetworkConnectionReconciler) notifyNickel(message string) {
	obj := n.conn.Object("com.github.shermp.nickeldbus", "/nickeldbus")
	call := obj.Call("com.github.shermp.nickeldbus.mwcToast", 0, 5000, "NextCloud Kobo Syncer", message)
	if call.Err != nil {
		log.Println("Failed to notify Nickel", call.Err)
	}
}

func checkNetwork() error {
	// Wait for the network to be fully connected
	for i := 0; i < 10; i++ {
		// Check if a web request to google is successful
		client := &http.Client{
			Timeout: 5 * time.Second,
		}
		req, err := http.NewRequest("GET", "http://www.google.com", nil)
		if err != nil {
			log.Println("Fatal error", err)
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
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