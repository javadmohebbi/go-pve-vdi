package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/ini.v1"
)

// HostInfo represents a Proxmox host configuration
type HostInfo struct {
	Host string
	Port int
}

// HostSet represents a group of Proxmox hosts with authentication settings
type HostSet struct {
	HostPool   []HostInfo
	Backend    string
	User       string
	TokenName  string
	TokenValue string
	TOTP       bool
	VerifySSL  bool
	PwResetCmd string
	AutoVMID   int
	KnockSeq   []interface{}
}

// Config represents the application configuration
type Config struct {
	Title          string
	Theme          string
	Icon           string
	Logo           string
	Kiosk          bool
	ViewerKiosk    bool
	Fullscreen     bool
	INIDebug       bool
	GuestType      string
	ShowReset      bool
	ShowHibernate  bool
	WindowWidth    int
	WindowHeight   int
	CurrentHostSet string
	Hosts          map[string]*HostSet
	SpiceProxyConv map[string]string
	AddlParams     map[string]string
}

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		Title:          "VDI Login",
		Theme:          "LightBlue",
		Kiosk:          false,
		ViewerKiosk:    true,
		Fullscreen:     true,
		INIDebug:       false,
		GuestType:      "both",
		ShowReset:      false,
		ShowHibernate:  false,
		WindowWidth:    0,
		WindowHeight:   0,
		CurrentHostSet: "DEFAULT",
		Hosts:          make(map[string]*HostSet),
		SpiceProxyConv: make(map[string]string),
		AddlParams:     make(map[string]string),
	}
}

// getDefaultConfigPaths returns the default configuration file paths based on OS
func getDefaultConfigPaths() []string {
	var paths []string

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		programFiles := os.Getenv("PROGRAMFILES")
		programFilesX86 := os.Getenv("PROGRAMFILES(x86)")

		paths = []string{
			filepath.Join(appData, "VDIClient", "vdiclient.ini"),
			filepath.Join(programFiles, "VDIClient", "vdiclient.ini"),
			filepath.Join(programFilesX86, "VDIClient", "vdiclient.ini"),
			"C:\\Program Files\\VDIClient\\vdiclient.ini",
		}
	} else {
		homeDir, _ := os.UserHomeDir()
		paths = []string{
			filepath.Join(homeDir, ".config", "VDIClient", "vdiclient.ini"),
			"/etc/vdiclient/vdiclient.ini",
			"/usr/local/etc/vdiclient/vdiclient.ini",
		}
	}

	return paths
}

// LoadConfig loads configuration from file or HTTP source
func LoadConfig(configLocation string, configType string, username string, password string, sslVerify bool) (*Config, error) {
	cfg := NewConfig()
	var iniCfg *ini.File
	var err error

	if configType == "file" {
		// Find config file
		if configLocation == "" {
			paths := getDefaultConfigPaths()
			for _, path := range paths {
				if _, err := os.Stat(path); err == nil {
					configLocation = path
					break
				}
			}
			if configLocation == "" {
				return nil, fmt.Errorf("unable to find configuration file in default locations")
			}
		} else {
			if _, err := os.Stat(configLocation); os.IsNotExist(err) {
				return nil, fmt.Errorf("configuration file does not exist: %s", configLocation)
			}
		}

		iniCfg, err = ini.Load(configLocation)
		if err != nil {
			return nil, fmt.Errorf("unable to read configuration file: %w", err)
		}
	} else if configType == "http" {
		if configLocation == "" {
			return nil, fmt.Errorf("--config_type http defined, yet no URL provided")
		}

		client := &http.Client{}
		req, err := http.NewRequest("GET", configLocation, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to create HTTP request: %w", err)
		}

		if username != "" && password != "" {
			req.SetBasicAuth(username, password)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch configuration from URL: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read HTTP response: %w", err)
		}

		iniCfg, err = ini.Load(body)
		if err != nil {
			return nil, fmt.Errorf("unable to parse configuration: %w", err)
		}
	}

	// Parse General section
	if iniCfg.HasSection("General") {
		general := iniCfg.Section("General")
		if general.HasKey("title") {
			cfg.Title = general.Key("title").String()
		}
		if general.HasKey("theme") {
			cfg.Theme = general.Key("theme").String()
		}
		if general.HasKey("icon") {
			iconPath := general.Key("icon").String()
			if _, err := os.Stat(iconPath); err == nil {
				cfg.Icon = iconPath
			}
		}
		if general.HasKey("logo") {
			logoPath := general.Key("logo").String()
			if _, err := os.Stat(logoPath); err == nil {
				cfg.Logo = logoPath
			}
		}
		if general.HasKey("kiosk") {
			cfg.Kiosk, _ = general.Key("kiosk").Bool()
		}
		if general.HasKey("viewer_kiosk") {
			cfg.ViewerKiosk, _ = general.Key("viewer_kiosk").Bool()
		}
		if general.HasKey("fullscreen") {
			cfg.Fullscreen, _ = general.Key("fullscreen").Bool()
		}
		if general.HasKey("inidebug") {
			cfg.INIDebug, _ = general.Key("inidebug").Bool()
		}
		if general.HasKey("guest_type") {
			cfg.GuestType = general.Key("guest_type").String()
		}
		if general.HasKey("show_reset") {
			cfg.ShowReset, _ = general.Key("show_reset").Bool()
		}
		if general.HasKey("show_hibernate") {
			cfg.ShowHibernate, _ = general.Key("show_hibernate").Bool()
		}
		if general.HasKey("window_width") {
			cfg.WindowWidth, _ = general.Key("window_width").Int()
		}
		if general.HasKey("window_height") {
			cfg.WindowHeight, _ = general.Key("window_height").Int()
		}
	} else {
		return nil, fmt.Errorf("no General section defined in configuration")
	}

	// Parse host configurations
	// Check for legacy Authentication section
	if iniCfg.HasSection("Authentication") && iniCfg.HasSection("Hosts") {
		// Legacy configuration format
		hostSet := &HostSet{
			Backend:    "pve",
			User:       "",
			VerifySSL:  true,
			HostPool:   []HostInfo{},
			KnockSeq:   []interface{}{},
		}

		auth := iniCfg.Section("Authentication")
		if auth.HasKey("auth_backend") {
			hostSet.Backend = auth.Key("auth_backend").String()
		}
		if auth.HasKey("user") {
			hostSet.User = auth.Key("user").String()
		}
		if auth.HasKey("token_name") {
			hostSet.TokenName = auth.Key("token_name").String()
		}
		if auth.HasKey("token_value") {
			hostSet.TokenValue = auth.Key("token_value").String()
		}
		if auth.HasKey("auth_totp") {
			hostSet.TOTP, _ = auth.Key("auth_totp").Bool()
		}
		if auth.HasKey("tls_verify") {
			hostSet.VerifySSL, _ = auth.Key("tls_verify").Bool()
		}
		if auth.HasKey("pwresetcmd") {
			hostSet.PwResetCmd = auth.Key("pwresetcmd").String()
		}
		if auth.HasKey("auto_vmid") {
			hostSet.AutoVMID, _ = auth.Key("auto_vmid").Int()
		}

		// Parse hosts
		hosts := iniCfg.Section("Hosts")
		for _, key := range hosts.KeyStrings() {
			port, _ := hosts.Key(key).Int()
			hostSet.HostPool = append(hostSet.HostPool, HostInfo{
				Host: key,
				Port: port,
			})
		}

		cfg.Hosts["DEFAULT"] = hostSet
	} else {
		// New style configuration with Hosts.* sections
		hostSections := iniCfg.SectionStrings()
		i := 0
		for _, section := range hostSections {
			if len(section) > 6 && section[:6] == "Hosts." {
				group := section[6:]
				if i == 0 {
					cfg.CurrentHostSet = group
				}

				hostSet := &HostSet{
					Backend:    "pve",
					User:       "",
					VerifySSL:  true,
					HostPool:   []HostInfo{},
					KnockSeq:   []interface{}{},
				}

				sec := iniCfg.Section(section)

				// Parse hostpool JSON
				if sec.HasKey("hostpool") {
					// For simplicity, we'll parse manually
					// In production, you'd use json.Unmarshal
					// For now, assume format: {"host1": port1, "host2": port2}
					// This is a simplified version - you may need to enhance this
				}

				if sec.HasKey("auth_backend") {
					hostSet.Backend = sec.Key("auth_backend").String()
				}
				if sec.HasKey("user") {
					hostSet.User = sec.Key("user").String()
				}
				if sec.HasKey("token_name") {
					hostSet.TokenName = sec.Key("token_name").String()
				}
				if sec.HasKey("token_value") {
					hostSet.TokenValue = sec.Key("token_value").String()
				}
				if sec.HasKey("auth_totp") {
					hostSet.TOTP, _ = sec.Key("auth_totp").Bool()
				}
				if sec.HasKey("tls_verify") {
					hostSet.VerifySSL, _ = sec.Key("tls_verify").Bool()
				}
				if sec.HasKey("pwresetcmd") {
					hostSet.PwResetCmd = sec.Key("pwresetcmd").String()
				}
				if sec.HasKey("auto_vmid") {
					hostSet.AutoVMID, _ = sec.Key("auto_vmid").Int()
				}

				cfg.Hosts[group] = hostSet
				i++
			}
		}
	}

	// Parse SpiceProxyRedirect section
	if iniCfg.HasSection("SpiceProxyRedirect") {
		spice := iniCfg.Section("SpiceProxyRedirect")
		for _, key := range spice.KeyStrings() {
			cfg.SpiceProxyConv[key] = spice.Key(key).String()
		}
	}

	// Parse AdditionalParameters section
	if iniCfg.HasSection("AdditionalParameters") {
		addl := iniCfg.Section("AdditionalParameters")
		for _, key := range addl.KeyStrings() {
			cfg.AddlParams[key] = addl.Key(key).String()
		}
	}

	return cfg, nil
}
