package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func runCLI(config *Config) {
	// Enable debug if configured
	SetDebug(config.Debug)

	fmt.Println("=== Proxmox VDI Client (CLI Mode) ===")
	fmt.Printf("Title: %s\n", config.Title)
	fmt.Printf("Current Host Set: %s\n", config.CurrentHostSet)
	fmt.Printf("Protocol: %s\n", strings.ToUpper(config.Protocol))
	if config.Debug {
		fmt.Println("Debug Mode: ENABLED")
	}
	fmt.Println()

	// Get host configuration
	hostSet := config.Hosts[config.CurrentHostSet]

	// Get credentials
	var username, password string

	if hostSet.User != "" {
		username = hostSet.User
		fmt.Printf("Username: %s (from config)\n", username)
	} else {
		fmt.Print("Username: ")
		reader := bufio.NewReader(os.Stdin)
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
	}

	// Check if using token authentication
	if hostSet.TokenName != "" && hostSet.TokenValue != "" {
		fmt.Println("Using API token authentication")
	} else {
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Failed to read password: %v", err)
		}
		password = string(passwordBytes)
		fmt.Println() // New line after password input
	}

	// Create Proxmox client
	proxmoxClient := NewProxmoxClient(config)

	// Authenticate
	fmt.Println("\n🔐 Authenticating...")
	connected, authenticated, err := proxmoxClient.Authenticate(username, password, "")

	if !connected {
		log.Fatalf("❌ Failed to connect to Proxmox: %v", err)
	}

	if !authenticated {
		log.Fatalf("❌ Authentication failed: %v", err)
	}

	fmt.Println("✅ Authentication successful!")
	fmt.Println()

	// Main loop
	for {
		// Get VMs
		fmt.Println("📋 Fetching VMs...")
		vms, err := proxmoxClient.GetVMs()
		if err != nil {
			log.Fatalf("❌ Failed to get VMs: %v", err)
		}

		if len(vms) == 0 {
			fmt.Println("No VMs found")
			return
		}

		fmt.Printf("✅ Found %d VM(s):\n\n", len(vms))

		// Display VMs
		fmt.Println("ID\tName\t\t\t\tNode\t\tType\tStatus")
		fmt.Println("──\t────\t\t\t\t────\t\t────\t──────")
		for i, vm := range vms {
			fmt.Printf("[%d]\t%-25s\t%-10s\t%s\t%s\n",
				i+1,
				truncateString(vm.Name, 25),
				vm.Node,
				vm.Type,
				vm.GetDisplayState())
		}

		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  1-N    : Select VM number to manage")
		fmt.Println("  r      : Refresh list")
		fmt.Println("  q      : Quit")
		fmt.Print("\nSelect: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "q" {
			fmt.Println("Goodbye!")
			return
		}

		if input == "r" {
			fmt.Println()
			continue
		}

		// Parse VM selection
		vmNum, err := strconv.Atoi(input)
		if err != nil || vmNum < 1 || vmNum > len(vms) {
			fmt.Println("❌ Invalid selection")
			fmt.Println()
			continue
		}

		selectedVM := vms[vmNum-1]
		manageVM(proxmoxClient, selectedVM, config)
	}
}

func manageVM(pc *ProxmoxClient, vm *VMInfo, config *Config) {
	for {
		fmt.Printf("\n=== Managing: %s (ID: %d) ===\n", vm.Name, vm.VMID)
		fmt.Printf("Node: %s\n", vm.Node)
		fmt.Printf("Type: %s\n", vm.Type)
		fmt.Printf("Status: %s\n", vm.GetDisplayState())
		fmt.Println()

		fmt.Println("Actions:")
		if vm.Status == "running" {
			fmt.Printf("  1 - Connect (%s)\n", strings.ToUpper(config.Protocol))
			fmt.Println("  2 - Stop VM")
			fmt.Println("  3 - Reset VM (Stop & Start)")
		} else {
			fmt.Println("  1 - Start VM")
		}
		fmt.Println("  b - Back to VM list")
		fmt.Println("  q - Quit")
		fmt.Print("\nSelect: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "q" {
			fmt.Println("Goodbye!")
			os.Exit(0)
		}

		if input == "b" {
			return
		}

		switch input {
		case "1":
			if vm.Status == "running" {
				// Connect
				fmt.Printf("\n🔌 Connecting to VM via %s...\n", strings.ToUpper(config.Protocol))
				err := ConnectToVM(pc, vm, config.Kiosk, config.ViewerKiosk, config.Fullscreen)
				if err != nil {
					fmt.Printf("❌ Connection failed: %v\n", err)
				} else {
					fmt.Printf("✅ %s viewer launched\n", strings.ToUpper(config.Protocol))
				}
			} else {
				// Start
				fmt.Println("\n▶️  Starting VM...")
				err := pc.StartVM(vm.Node, vm.VMID, vm.Type)
				if err != nil {
					fmt.Printf("❌ Failed to start VM: %v\n", err)
				} else {
					fmt.Println("✅ VM started successfully")
					vm.Status = "running"
				}
			}

		case "2":
			if vm.Status == "running" {
				fmt.Println("\n⏹️  Stopping VM...")
				err := pc.StopVM(vm.Node, vm.VMID, vm.Type)
				if err != nil {
					fmt.Printf("❌ Failed to stop VM: %v\n", err)
				} else {
					fmt.Println("✅ VM stopped successfully")
					vm.Status = "stopped"
				}
			} else {
				fmt.Println("❌ Invalid action for current state")
			}

		case "3":
			if vm.Status == "running" {
				fmt.Println("\n🔄 Resetting VM (Stop & Start)...")
				err := ResetVM(pc, vm)
				if err != nil {
					fmt.Printf("❌ Failed to reset VM: %v\n", err)
				} else {
					fmt.Println("✅ VM reset successfully")
					vm.Status = "running"
				}
			} else {
				fmt.Println("❌ Invalid action for current state")
			}

		default:
			fmt.Println("❌ Invalid selection")
		}

		fmt.Println("\nPress Enter to continue...")
		reader.ReadString('\n')
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
