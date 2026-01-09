package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
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

	DebugLog("Starting authentication for user: %s", username)
	DebugLog("Host pool size: %d", len(hostSet.HostPool))

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
		DebugLog("Attempting connection to %s:%d", hostInfo.Host, hostInfo.Port)
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
			DebugLog("Using API token authentication")
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
				DebugLog("API token authentication successful")
			} else {
				DebugLog("API token authentication failed: %v", err)
			}
		} else {
			// Username/Password authentication
			DebugLog("Using username/password authentication")
			credentials := &proxmox.Credentials{
				Username: fmt.Sprintf("%s@%s", username, hostSet.Backend),
				Password: password,
			}

			// Add OTP if provided
			if totp != "" {
				DebugLog("TOTP provided")
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
				DebugLog("Password authentication successful")
			} else {
				DebugLog("Password authentication failed: %v", err)
			}
		}

		if connected && authenticated {
			DebugLog("Successfully authenticated to %s:%d", hostInfo.Host, hostInfo.Port)
			return true, true, nil
		}

		if connected && !authenticated {
			DebugLog("Connected but authentication failed")
			return true, false, err
		}

		DebugLog("Connection failed to %s:%d - trying next host", hostInfo.Host, hostInfo.Port)
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

	// Get all VMs
	DebugLog("Fetching cluster resources (type: vm)")
	resources, err := cluster.Resources(ctx, "vm")
	if err != nil {
		return nil, fmt.Errorf("unable to get VMs: %w", err)
	}

	DebugLog("Found %d total resources", len(resources))

	for _, resource := range resources {
		DebugLog("Resource: VMID=%d Name=%s Node=%s Type=%s Status=%s Template=%d",
			resource.VMID, resource.Name, resource.Node, resource.Type, resource.Status, resource.Template)

		// Skip templates only
		if resource.Template == 1 {
			DebugLog("  Skipping (template)")
			continue
		}

		// Determine VM type
		vmType := "qemu"
		if resource.Type == "lxc" {
			vmType = "lxc"
		}

		vm := &VMInfo{
			VMID:   int(resource.VMID),
			Name:   resource.Name,
			Node:   resource.Node,
			Type:   vmType,
			Status: resource.Status,
			Lock:   "",
		}

		vms = append(vms, vm)
		DebugLog("  Added to list")
	}

	DebugLog("Returning %d VMs", len(vms))
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

	if vmType != "qemu" {
		return nil, fmt.Errorf("SPICE only supported for QEMU VMs, not %s", vmType)
	}

	ctx := context.Background()
	DebugLog("Getting SPICE config for VM %d on node %s", vmID, vmNode)

	// Make direct API call to get SPICE proxy configuration
	// POST /api2/json/nodes/{node}/qemu/{vmid}/spiceproxy
	url := fmt.Sprintf("/nodes/%s/qemu/%d/spiceproxy", vmNode, vmID)

	DebugLog("Calling SPICE proxy API: %s", url)

	// Use the client's internal request method
	var result map[string]interface{}
	err := pc.Client.Post(ctx, url, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get SPICE proxy config: %w", err)
	}

	DebugLog("SPICE proxy response: %+v", result)

	// Convert result to string map
	spiceConfig := make(map[string]string)
	for key, value := range result {
		spiceConfig[key] = fmt.Sprintf("%v", value)
	}

	DebugLog("SpiceProxyConv map has %d entries", len(pc.Config.SpiceProxyConv))
	for k, v := range pc.Config.SpiceProxyConv {
		DebugLog("  Map entry: '%s' => '%s'", k, v)
	}

	// Handle proxy redirection if configured
	if proxyVal, ok := spiceConfig["proxy"]; ok {
		DebugLog("Original proxy from API: '%s'", proxyVal)
		// Check if we need to convert the proxy hostname
		if strings.HasPrefix(proxyVal, "http://") {
			hostname := strings.ToLower(proxyVal[7:])

			// Try exact match first
			if converted, exists := pc.Config.SpiceProxyConv[hostname]; exists {
				spiceConfig["proxy"] = fmt.Sprintf("http://%s", converted)
				DebugLog("Converted proxy to: %s (exact match)", spiceConfig["proxy"])
			} else {
				// Try without port, then add port back
				parts := strings.Split(hostname, ":")
				if len(parts) == 2 {
					hostOnly := parts[0]
					port := parts[1]

					// Check if we have a conversion for hostname:port
					hostPort := fmt.Sprintf("%s:%s", hostOnly, port)
					if converted, exists := pc.Config.SpiceProxyConv[hostPort]; exists {
						spiceConfig["proxy"] = fmt.Sprintf("http://%s", converted)
						DebugLog("Converted proxy to: %s (host:port match)", spiceConfig["proxy"])
					} else if converted, exists := pc.Config.SpiceProxyConv[hostOnly]; exists {
						// Conversion found for hostname only, add port back
						spiceConfig["proxy"] = fmt.Sprintf("http://%s:%s", converted, port)
						DebugLog("Converted proxy to: %s (hostname match, port preserved)", spiceConfig["proxy"])
					}
				}
			}
		}
	}

	// Add additional parameters from config
	for key, value := range pc.Config.AddlParams {
		spiceConfig[key] = value
		DebugLog("Added additional param: %s=%s", key, value)
	}

	DebugLog("Final SPICE config: %+v", spiceConfig)
	return spiceConfig, nil
}
