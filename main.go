package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	// Command line flags
	configType := flag.String("config_type", "file", "Select config type (file or http)")
	configLocation := flag.String("config_location", "", "Specify the config location")
	configUsername := flag.String("config_username", "", "HTTP basic authentication username")
	configPassword := flag.String("config_password", "", "HTTP basic authentication password")
	ignoreSsl := flag.Bool("ignore_ssl", false, "HTTPS ignore SSL certificate errors")

	flag.Parse()

	// Invert the SSL verify logic (ignoreSsl = true means verify = false)
	sslVerify := !*ignoreSsl

	// Load configuration
	config, err := LoadConfig(*configLocation, *configType, *configUsername, *configPassword, sslVerify)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Check if virt-viewer is available
	_, err = FindVirtViewerCommand()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create and run application
	appState := NewAppState(config)
	appState.Run()
}
