package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	templateDir = "./public/templates"
)

var (
	fileDir string
)

func init() {
	flag.StringVar(&fileDir, "files", "", "The directory to search for files in")
}

func validateArgs() {
	if fileDir == "" {
		log.Fatalf("A file directory must be provided.")
	}
}

func main() {
	flag.Parse()
	validateArgs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := NewFileServer(":8080", fileDir, templateDir)

	// Enforce clean shutdown on SIGTERM and SIGINT
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-signals:
			log.Printf("Shutting down file server...")
			srv.Stop(ctx)
			cancel()
		case <-ctx.Done():
		}
	}()

	err := srv.Start()
	if err != nil {
		log.Printf("%v", err)
	}
}
