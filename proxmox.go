package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	proxmox "github.com/luthermonson/go-proxmox"
)

// ProxmoxClient wraps the Proxmox API client
type ProxmoxClient struct {
	Client *proxmox.Client
	Config *Config
}

// NewProxmoxClient creates a new Proxmox client instance
func NewProxmoxClient(cfg *Config) *ProxmoxClient {
	return &ProxmoxClient{
		Config: cfg,
	}
}

// Authenticate connects and authenticates to a Proxmox server
func (pc *ProxmoxClient) Authenticate(username, password, totp string) (bool, bool, error) {
	hostSet := pc.Config.Hosts[pc.Config.CurrentHostSet]

	// Shuffle host pool to distribute load
	rand.Seed(time.Now().UnixNano())
	hostPool := make([]HostInfo, len(hostSet.HostPool))
	copy(hostPool, hostSet.HostPool)
	rand.Shuffle(len(hostPool), func(i, j int) {
		hostPool[i], hostPool[j] = hostPool[j], hostPool[i]
	})

	var lastErr error
	ctx := context.Background()

	// Try each host in the pool
	for _, hostInfo := range hostPool {
		connected := false
		authenticated := false

		// Prepare HTTP client with SSL verification settings
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: !hostSet.VerifySSL,
				},
			},
		}

		url := fmt.Sprintf("https://%s:%d/api2/json", hostInfo.Host, hostInfo.Port)

		var err error

		// Authenticate based on credentials type
		if hostSet.TokenName != "" && hostSet.TokenValue != "" {
			// API Token authentication
			tokenStr := fmt.Sprintf("%s=%s", hostSet.TokenName, hostSet.TokenValue)
			userID := fmt.Sprintf("%s@%s", username, hostSet.Backend)

			pc.Client = proxmox.NewClient(url,
				proxmox.WithHTTPClient(httpClient),
				proxmox.WithAPIToken(userID, tokenStr),
			)

			// Test connection
			_, err = pc.Client.Version(ctx)
			if err == nil {
				connected = true
				authenticated = true
			}
		} else {
			// Username/Password authentication
			credentials := &proxmox.Credentials{
				Username: fmt.Sprintf("%s@%s", username, hostSet.Backend),
				Password: password,
			}

			// Add OTP if provided
			if totp != "" {
				credentials.Otp = totp
			}

			pc.Client = proxmox.NewClient(url,
				proxmox.WithHTTPClient(httpClient),
				proxmox.WithCredentials(credentials),
			)

			// Test connection
			_, err = pc.Client.Version(ctx)
			if err == nil {
				connected = true
				authenticated = true
			}
		}

		if connected && authenticated {
			return true, true, nil
		}

		if connected && !authenticated {
			return true, false, err
		}

		lastErr = err
	}

	// Failed to connect to any host
	return false, false, lastErr
}

// GetVMs retrieves all VMs from the Proxmox cluster
func (pc *ProxmoxClient) GetVMs() ([]*VMInfo, error) {
	if pc.Client == nil {
		return nil, fmt.Errorf("not connected to Proxmox")
	}

	ctx := context.Background()
	vms := []*VMInfo{}

	// Get cluster resources
	cluster, err := pc.Client.Cluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get cluster info: %w", err)
	}

	// Get all nodes - cluster.Nodes is already a slice, not a function
	onlineNodes := make(map[string]bool)
	for _, node := range cluster.Nodes {
		if node.Status == "online" {
			onlineNodes[node.Node] = true
		}
	}

	// Get all VMs
	resources, err := cluster.Resources(ctx, "vm")
	if err != nil {
		return nil, fmt.Errorf("unable to get VMs: %w", err)
	}

	for _, resource := range resources {
		// Skip VMs on offline nodes
		if !onlineNodes[resource.Node] {
			continue
		}

		// Skip templates
		if resource.Template == 1 {
			continue
		}

		// Filter by guest type if configured
		vmType := "qemu"
		if resource.Type == "lxc" {
			vmType = "lxc"
		}

		if pc.Config.GuestType != "both" && pc.Config.GuestType != vmType {
			continue
		}

		vm := &VMInfo{
			VMID:   int(resource.VMID),
			Name:   resource.Name,
			Node:   resource.Node,
			Type:   vmType,
			Status: resource.Status,
			Lock:   "", // ClusterResource doesn't have Lock field
		}

		vms = append(vms, vm)
	}

	return vms, nil
}

// StartVM starts a virtual machine
func (pc *ProxmoxClient) StartVM(vmNode string, vmID int, vmType string) error {
	if pc.Client == nil {
		return fmt.Errorf("not connected to Proxmox")
	}

	ctx := context.Background()

	node, err := pc.Client.Node(ctx, vmNode)
	if err != nil {
		return fmt.Errorf("unable to get node: %w", err)
	}

	if vmType == "qemu" {
		vm, err := node.VirtualMachine(ctx, vmID)
		if err != nil {
			return fmt.Errorf("unable to get VM: %w", err)
		}

		task, err := vm.Start(ctx)
		if err != nil {
			return fmt.Errorf("unable to start VM: %w", err)
		}

		// Wait for task completion
		err = task.WaitFor(ctx, 30)
		if err != nil {
			return fmt.Errorf("error waiting for VM to start: %w", err)
		}

		if !task.IsCompleted || task.ExitStatus != "OK" {
			return fmt.Errorf("failed to start VM: %s", task.ExitStatus)
		}

		return nil
	} else {
		// LXC container
		container, err := node.Container(ctx, vmID)
		if err != nil {
			return fmt.Errorf("unable to get container: %w", err)
		}

		task, err := container.Start(ctx)
		if err != nil {
			return fmt.Errorf("unable to start container: %w", err)
		}

		// Wait for task completion
		err = task.WaitFor(ctx, 30)
		if err != nil {
			return fmt.Errorf("error waiting for container to start: %w", err)
		}

		if !task.IsCompleted || task.ExitStatus != "OK" {
			return fmt.Errorf("failed to start container: %s", task.ExitStatus)
		}

		return nil
	}
}

// StopVM stops a virtual machine
func (pc *ProxmoxClient) StopVM(vmNode string, vmID int, vmType string) error {
	if pc.Client == nil {
		return fmt.Errorf("not connected to Proxmox")
	}

	ctx := context.Background()

	node, err := pc.Client.Node(ctx, vmNode)
	if err != nil {
		return fmt.Errorf("unable to get node: %w", err)
	}

	if vmType == "qemu" {
		vm, err := node.VirtualMachine(ctx, vmID)
		if err != nil {
			return fmt.Errorf("unable to get VM: %w", err)
		}

		task, err := vm.Stop(ctx)
		if err != nil {
			return fmt.Errorf("unable to stop VM: %w", err)
		}

		// Wait for task completion
		err = task.WaitFor(ctx, 30)
		if err != nil {
			return fmt.Errorf("error waiting for VM to stop: %w", err)
		}

		if !task.IsCompleted || task.ExitStatus != "OK" {
			return fmt.Errorf("failed to stop VM: %s", task.ExitStatus)
		}

		return nil
	} else {
		// LXC container
		container, err := node.Container(ctx, vmID)
		if err != nil {
			return fmt.Errorf("unable to get container: %w", err)
		}

		task, err := container.Stop(ctx)
		if err != nil {
			return fmt.Errorf("unable to stop container: %w", err)
		}

		// Wait for task completion
		err = task.WaitFor(ctx, 30)
		if err != nil {
			return fmt.Errorf("error waiting for container to stop: %w", err)
		}

		if !task.IsCompleted || task.ExitStatus != "OK" {
			return fmt.Errorf("failed to stop container: %s", task.ExitStatus)
		}

		return nil
	}
}

// GetSPICEConfig gets SPICE proxy configuration for a VM
func (pc *ProxmoxClient) GetSPICEConfig(vmNode string, vmID int, vmType string) (map[string]string, error) {
	if pc.Client == nil {
		return nil, fmt.Errorf("not connected to Proxmox")
	}

	ctx := context.Background()

	node, err := pc.Client.Node(ctx, vmNode)
	if err != nil {
		return nil, fmt.Errorf("unable to get node: %w", err)
	}

	// Get VM
	if vmType == "qemu" {
		vm, err := node.VirtualMachine(ctx, vmID)
		if err != nil {
			return nil, fmt.Errorf("unable to get VM: %w", err)
		}

		// Get SPICE proxy - this may require a direct API call if not in the library
		// For now, we'll construct the basic config
		spiceConfig := make(map[string]string)

		// Call the SPICE proxy endpoint
		// This is a workaround since the library might not have direct support
		// You may need to use vm's underlying client to make a raw API call
		_ = vm // Use vm to avoid unused variable error

		// Placeholder - in production you'd make: POST /nodes/{node}/qemu/{vmid}/spiceproxy
		// and parse the returned config
		spiceConfig["type"] = "spice"
		spiceConfig["host"] = vmNode
		spiceConfig["port"] = "3128"

		return spiceConfig, fmt.Errorf("SPICE proxy requires manual API call - not fully implemented yet")
	}

	return nil, fmt.Errorf("SPICE only supported for QEMU VMs")
}
