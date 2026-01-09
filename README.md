# Proxmox VDI Client (Go)

A modern VDI (Virtual Desktop Infrastructure) client for Proxmox VE, written in Go with Fyne GUI framework.

## Features

- **User-friendly GUI**: Built with Fyne for a native look and feel across platforms
- **Proxmox Integration**: Full integration with Proxmox VE API
- **SPICE Protocol**: Connect to VMs using the SPICE protocol via virt-viewer
- **Multiple Authentication**: Support for username/password, API tokens, and TOTP/2FA
- **VM Management**: Start, stop, and reset virtual machines
- **Multiple Host Support**: Connect to multiple Proxmox clusters
- **Flexible Configuration**: INI file or HTTP-based configuration
- **Cross-Platform**: Works on Linux, Windows, and macOS

## Requirements

### Runtime Requirements
- **virt-viewer**: Required for SPICE connections
  - Linux: `apt install virt-viewer` or `yum install virt-viewer`
  - Windows: Download from https://virt-manager.org/download/
  - macOS: `brew install virt-viewer`

### Build Requirements
- Go 1.20 or later
- Fyne dependencies (handled automatically by Go modules)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/javadmohebbi/go-pve-vdi.git
cd go-pve-vdi

# Build the application
go build -o go-pve-vdi

# Run the application
./go-pve-vdi
```

### Building for Different Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o go-pve-vdi-linux

# Windows
GOOS=windows GOARCH=amd64 go build -o go-pve-vdi.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -o go-pve-vdi-macos
```

## Configuration

Create a configuration file at one of the following locations:

**Linux:**
- `~/.config/VDIClient/vdiclient.ini`
- `/etc/vdiclient/vdiclient.ini`
- `/usr/local/etc/vdiclient/vdiclient.ini`

**Windows:**
- `%APPDATA%\VDIClient\vdiclient.ini`
- `%PROGRAMFILES%\VDIClient\vdiclient.ini`

**Example configurations:**

See the following files for configuration examples:
- `config.example` - 10 complete configuration examples for different use cases
- `vdiclient.ini.example` - Detailed configuration template with all options

Minimal configuration:

```ini
[General]
title=My VDI

[Authentication]
auth_backend=pve
auth_totp=false
tls_verify=true

[Hosts]
proxmox.example.com=8006
```

## Usage

### Basic Usage

```bash
# Use default configuration file location
./go-pve-vdi

# Specify configuration file
./go-pve-vdi -config_location /path/to/config.ini

# Use HTTP configuration
./go-pve-vdi -config_type http -config_location https://config.example.com/vdiclient.ini

# HTTP configuration with basic auth
./go-pve-vdi -config_type http -config_location https://config.example.com/vdiclient.ini \
  -config_username myuser -config_password mypass

# Ignore SSL certificate errors
./go-pve-vdi -ignore_ssl
```

### Command Line Options

- `-config_type`: Configuration type (`file` or `http`), default: `file`
- `-config_location`: Path to configuration file or HTTP URL
- `-config_username`: HTTP basic authentication username
- `-config_password`: HTTP basic authentication password
- `-ignore_ssl`: Ignore SSL certificate verification errors

## Configuration Options

### General Section

- `title`: Window title (default: "VDI Login")
- `theme`: GUI theme (currently unused)
- `icon`: Path to icon file
- `logo`: Path to logo image
- `kiosk`: Enable kiosk mode (hide window decorations)
- `viewer_kiosk`: Launch viewer in kiosk mode
- `fullscreen`: Launch viewer in fullscreen
- `guest_type`: Filter VMs by type: `both`, `qemu`, or `lxc`
- `show_reset`: Show reset button for VMs
- `show_hibernate`: Show hibernate button for VMs
- `window_width`: Window width in pixels
- `window_height`: Window height in pixels

### Authentication Section

- `auth_backend`: Authentication backend (usually `pve` or `pam`)
- `user`: Default username (leave empty to prompt)
- `token_name`: API token name (for token authentication)
- `token_value`: API token value
- `auth_totp`: Enable TOTP/2FA
- `tls_verify`: Verify SSL certificates (default: true)
- `pwresetcmd`: Command to run for password reset
- `auto_vmid`: Auto-connect to specific VM ID

### Hosts Section

List Proxmox hosts and ports:

```ini
[Hosts]
host1.example.com=8006
host2.example.com=8006
```

For multiple host groups, use the new format:

```ini
[Hosts.Production]
hostpool={"host1.example.com": 8006, "host2.example.com": 8006}
auth_backend=pve

[Hosts.Development]
hostpool={"dev-host.example.com": 8006}
auth_backend=pve
tls_verify=false
```

## Development

### Project Structure

- `main.go`: Application entry point and command-line argument handling
- `config.go`: Configuration loading and parsing
- `proxmox.go`: Proxmox API client wrapper
- `vm.go`: VM operations and SPICE connection handling
- `gui.go`: Fyne GUI implementation

### Building and Testing

```bash
# Run the application in development mode
go run .

# Build for production
go build -ldflags="-s -w" -o go-pve-vdi

# Run tests (if available)
go test ./...
```

## Differences from Python Version

This Go implementation maintains feature parity with the original Python version while offering:

- **Better Performance**: Compiled binary with faster startup time
- **Easier Deployment**: Single binary with no Python dependencies
- **Modern GUI**: Fyne framework for native look and feel
- **Type Safety**: Go's static typing for more robust code
- **Better Concurrency**: Native goroutines for responsive UI

## License

This project is provided as-is for educational and personal use.

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.

## Author

Javad Mohebbi (https://github.com/javadmohebbi)

## Original Python Version

This is a rewrite of the original Python VDI client. The original version used:
- Python 3
- Proxmoxer library
- PySimpleGUI

This Go version provides the same functionality with improved performance and easier deployment.
