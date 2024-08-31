package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aleskandro/nextcloud-kobo-synchronizer/pkg"
)

func main() {
	configFilePath := flag.String("config-file", "", "The path to the yaml config file")
	basePath := flag.String("base-path", "", "The base path to use for relative paths in the config file")
	sync := flag.Bool("sync", false, "Run the syncer at startup")
	flag.Parse()
	config, err := pkg.LoadConfig(*configFilePath, *basePath)
	if err != nil {
		log.Println("NextCloud Kobo syncer failed at loading config")
		log.Println(err)
		return
	}
	controller := pkg.NewNetworkConnectionReconciler(config)
	ctx := SetupSignalHandler()
	if *sync {
		controller.HandleWmNetworkConnected(ctx)
	}
	controller.Run(ctx)
}

// Gently stolen from the k8s source code
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
var onlyOneSignalHandler = make(chan struct{})

// SetupSignalHandler registers for SIGTERM and SIGINT. A context is returned
// which is canceled on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func SetupSignalHandler() context.Context {
	close(onlyOneSignalHandler) // panics when called twice

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		cancel()
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return ctx
}
