package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// VMInfo represents a Virtual Machine or Container
type VMInfo struct {
	VMID   int
	Name   string
	Node   string
	Type   string // "qemu" or "lxc"
	Status string
	Lock   string
}

// GetDisplayState returns a human-readable state for the VM
func (vm *VMInfo) GetDisplayState() string {
	if vm.Status != "running" {
		return "stopped"
	}

	if vm.Lock != "" {
		if vm.Lock == "suspending" || vm.Lock == "suspended" {
			if vm.Lock == "suspended" {
				return "starting"
			}
			return vm.Lock
		}
		return vm.Lock
	}

	return vm.Status
}

// IsConnectable returns whether the VM can be connected to
func (vm *VMInfo) IsConnectable() bool {
	state := vm.GetDisplayState()
	return state == "running"
}

// FindVirtViewerCommand finds the virt-viewer executable path
func FindVirtViewerCommand() (string, error) {
	if runtime.GOOS == "windows" {
		// On Windows, look for remote-viewer.exe in registry
		cmd := exec.Command("cmd", "/c", "ftype VirtViewer.vvfile")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("virt-viewer not found. Please install from https://virt-manager.org/download/")
		}

		// Parse the output to get the executable path
		parts := strings.Split(string(output), "=")
		if len(parts) < 2 {
			return "", fmt.Errorf("unable to parse virt-viewer path")
		}

		// Extract the first quoted part or first word
		cmdPart := strings.TrimSpace(parts[1])
		if strings.HasPrefix(cmdPart, "\"") {
			endQuote := strings.Index(cmdPart[1:], "\"")
			if endQuote > 0 {
				return cmdPart[1 : endQuote+1], nil
			}
		}

		fields := strings.Fields(cmdPart)
		if len(fields) > 0 {
			return fields[0], nil
		}

		return "", fmt.Errorf("unable to extract virt-viewer executable path")
	} else {
		// On Linux/macOS, use which command
		cmd := exec.Command("which", "remote-viewer")
		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf("virt-viewer not found. Please install using: apt install virt-viewer")
		}
		return "remote-viewer", nil
	}
}

// ConnectToVM starts the VM if needed and connects via SPICE
func ConnectToVM(pc *ProxmoxClient, vm *VMInfo, kiosk bool, viewerKiosk bool, fullscreen bool) error {
	// Check if VM is running, start if not
	if vm.Status != "running" {
		err := pc.StartVM(vm.Node, vm.VMID, vm.Type)
		if err != nil {
			return fmt.Errorf("unable to start VM: %w", err)
		}
	}

	// Get SPICE configuration
	spiceConfig, err := pc.GetSPICEConfig(vm.Node, vm.VMID, vm.Type)
	if err != nil {
		return fmt.Errorf("unable to get SPICE configuration: %w\nIs SPICE display configured for your VM?", err)
	}

	// Build virt-viewer configuration
	var configBuffer bytes.Buffer
	configBuffer.WriteString("[virt-viewer]\n")

	// Process SPICE config
	for key, value := range spiceConfig {
		if key == "proxy" {
			// Handle proxy conversion
			if strings.HasPrefix(value, "http://") {
				val := strings.ToLower(value[7:])
				if converted, ok := pc.Config.SpiceProxyConv[val]; ok {
					configBuffer.WriteString(fmt.Sprintf("%s=http://%s\n", key, converted))
				} else {
					configBuffer.WriteString(fmt.Sprintf("%s=%s\n", key, value))
				}
			} else {
				configBuffer.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			}
		} else {
			configBuffer.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
	}

	// Add additional parameters
	for key, value := range pc.Config.AddlParams {
		configBuffer.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	// Find virt-viewer command
	vvCmd, err := FindVirtViewerCommand()
	if err != nil {
		return err
	}

	// Build command arguments
	args := []string{}
	if kiosk && viewerKiosk {
		args = append(args, "--kiosk", "--kiosk-quit", "on-disconnect")
	} else if fullscreen {
		args = append(args, "--full-screen")
	}
	args = append(args, "-") // Read from stdin

	// Execute virt-viewer
	cmd := exec.Command(vvCmd, args...)
	cmd.Stdin = &configBuffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("unable to start virt-viewer: %w", err)
	}

	return nil
}

// ResetVM stops and then starts a VM
func ResetVM(pc *ProxmoxClient, vm *VMInfo) error {
	// Stop the VM
	err := pc.StopVM(vm.Node, vm.VMID, vm.Type)
	if err != nil {
		return fmt.Errorf("unable to stop VM: %w", err)
	}

	// Start the VM
	err = pc.StartVM(vm.Node, vm.VMID, vm.Type)
	if err != nil {
		return fmt.Errorf("unable to start VM after reset: %w", err)
	}

	return nil
}
