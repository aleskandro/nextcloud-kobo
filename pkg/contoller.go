package pkg

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

type NetworkConnectionReconciler struct {
	conn          *dbus.Conn
	dbusChan      chan *dbus.Signal
	config        *Config
	toastsChan    chan string
	wg            *sync.WaitGroup
	syncCtx       context.Context
	syncCtxCancel context.CancelFunc
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
		conn:       conn,
		dbusChan:   ch,
		config:     config,
		toastsChan: make(chan string, 16),
	}
}

func (n *NetworkConnectionReconciler) Run(ctx context.Context) {
	defer func() {
		fmt.Println("Exiting network connection reconciler")
		n.conn.RemoveSignal(n.dbusChan)
		close(n.dbusChan)
		//nolint:errcheck
		n.conn.Close()
	}()
	for {
		fmt.Println("Listening for network connection signals from Nickel...")
		select {
		case <-ctx.Done():
			fmt.Println("Context done")
			return
		case signal, ok := <-n.dbusChan:
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
			n.HandleWmNetworkConnected(ctx)
		}
	}
}

func (n *NetworkConnectionReconciler) HandleWmNetworkConnected(ctx context.Context) {
	if n.syncCtxCancel != nil {
		n.syncCtxCancel()
	}
	n.wg.Wait()
	n.syncCtx, n.syncCtxCancel = context.WithCancel(ctx)
	n.wg.Add(3)
	go func() {
		defer n.wg.Done()
		n.keepNetworkAlive(n.syncCtx)
	}()
	go func() {
		defer n.wg.Done()
		n.dispatchMessages(n.syncCtx)
	}()
	go func() {
		defer n.wg.Done()
		defer func() {
			if n.syncCtx.Err() == nil {
				n.syncCtxCancel()
				n.syncCtxCancel = nil
			}
		}()
		n.sync(ctx)
		if n.config.AutoUpdate {
			n.updateNow()
		}
	}()
}

func (n *NetworkConnectionReconciler) keepNetworkAlive(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			obj := n.conn.Object("com.github.shermp.nickeldbus", "/nickeldbus")
			call := obj.Call("com.github.shermp.nickeldbus.wfmConnectWirelessSilently", 0)
			if call.Err != nil {
				log.Println("Failed to notify Nickel", call.Err)
			}
		}
	}
}

func (n *NetworkConnectionReconciler) dispatchMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-n.toastsChan:
			n.notifyNickel(message)
		}
	}
}

func (n *NetworkConnectionReconciler) rescanBooks() {
	obj := n.conn.Object("com.github.shermp.nickeldbus", "/nickeldbus")
	call := obj.Call("com.github.shermp.nickeldbus.pfmRescanBooks", 0)
	if call.Err != nil {
		log.Println("Failed to rescan books", call.Err)
	}
}

func (n *NetworkConnectionReconciler) notifyNickel(message string) {
	obj := n.conn.Object("com.github.shermp.nickeldbus", "/nickeldbus")
	call := obj.Call("com.github.shermp.nickeldbus.mwcToast", 0, 5000, "NextCloud Kobo Syncer", message)
	if call.Err != nil {
		log.Println("Failed to notify Nickel", call.Err)
	}
	time.Sleep(time.Second * 5)
}
