package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define all flags upfront
	cliMode := flag.Bool("cli", false, "Run in CLI mode (no GUI)")
	configType := flag.String("config_type", "file", "Select config type (file or http)")
	configLocation := flag.String("config_location", "", "Specify the config location")
	configUsername := flag.String("config_username", "", "HTTP basic authentication username")
	configPassword := flag.String("config_password", "", "HTTP basic authentication password")
	ignoreSsl := flag.Bool("ignore_ssl", false, "HTTPS ignore SSL certificate errors")

	// Parse once
	flag.Parse()

	// Invert the SSL verify logic
	sslVerify := !*ignoreSsl

	// Load configuration
	config, err := LoadConfig(*configLocation, *configType, *configUsername, *configPassword, sslVerify)
	if err != nil {
		os.Exit(1)
	}

	if *cliMode {
		// Run CLI version
		runCLI(config)
	} else {
		// Run GUI version
		// appState := NewAppState(config)
		// appState.Run()
		fmt.Println("Not Implemented! run with '-cli' to run CLI interface")
	}
}
