package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Application state
type AppState struct {
	App           fyne.App
	Window        fyne.Window
	Config        *Config
	ProxmoxClient *ProxmoxClient
	LoggedIn      bool
}

// NewAppState creates a new application state
func NewAppState(cfg *Config) *AppState {
	return &AppState{
		App:           app.New(),
		Config:        cfg,
		ProxmoxClient: NewProxmoxClient(cfg),
		LoggedIn:      false,
	}
}

// ShowLoginWindow displays the login window
func (as *AppState) ShowLoginWindow() bool {
	hostSet := as.Config.Hosts[as.Config.CurrentHostSet]

	// Check if we can skip login (token authentication)
	if hostSet.User != "" && hostSet.TokenName != "" && hostSet.TokenValue != "" && len(as.Config.Hosts) == 1 {
		// Auto-login with token
		connected, authenticated, err := as.ProxmoxClient.Authenticate(hostSet.User, "", "")
		if !connected {
			dialog.ShowError(fmt.Errorf("unable to connect to any VDI server: %w", err), as.Window)
			return false
		}
		if !authenticated {
			dialog.ShowError(fmt.Errorf("authentication failed"), as.Window)
			return false
		}
		as.LoggedIn = true
		return true
	}

	// Create login window
	as.Window = as.App.NewWindow(as.Config.Title)

	// Username field
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Username")
	if hostSet.User != "" {
		usernameEntry.SetText(hostSet.User)
		if hostSet.TokenName != "" && hostSet.TokenValue != "" {
			usernameEntry.Disable()
		}
	}

	// Password field
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")
	if hostSet.TokenName != "" && hostSet.TokenValue != "" {
		passwordEntry.Disable()
	}

	// OTP field (if TOTP is enabled)
	var otpEntry *widget.Entry
	var otpContainer *fyne.Container
	if hostSet.TOTP {
		otpEntry = widget.NewEntry()
		otpEntry.SetPlaceHolder("OTP Key")
		otpContainer = container.NewVBox(
			widget.NewLabel("OTP Key"),
			otpEntry,
		)
	}

	// Server group selector (if multiple groups)
	var groupSelector *widget.Select
	if len(as.Config.Hosts) > 1 {
		groups := []string{}
		for key := range as.Config.Hosts {
			groups = append(groups, key)
		}
		groupSelector = widget.NewSelect(groups, func(selected string) {
			as.Config.CurrentHostSet = selected
			as.Window.Close()
			as.ShowLoginWindow()
		})
		groupSelector.SetSelected(as.Config.CurrentHostSet)
	}

	// Login button
	loginButton := widget.NewButton("Log In", func() {
		username := usernameEntry.Text
		password := passwordEntry.Text
		totp := ""
		if otpEntry != nil {
			totp = otpEntry.Text
		}

		// Show progress dialog
		progress := dialog.NewCustomWithoutButtons("Authenticating", widget.NewLabel("Please wait, authenticating..."), as.Window)
		progress.Show()

		// Authenticate
		go func() {
			connected, authenticated, err := as.ProxmoxClient.Authenticate(username, password, totp)
			progress.Hide()

			if !connected {
				dialog.ShowError(fmt.Errorf("unable to connect to any VDI server: %w", err), as.Window)
				return
			}
			if !authenticated {
				dialog.ShowError(fmt.Errorf("invalid username and/or password"), as.Window)
				return
			}

			as.LoggedIn = true
			as.Window.Close()
		}()
	})

	// Cancel button
	cancelButton := widget.NewButton("Cancel", func() {
		as.Window.Close()
	})

	// Build layout
	content := container.NewVBox()

	// Add title/logo
	titleLabel := widget.NewLabel(as.Config.Title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter
	content.Add(titleLabel)

	// Add group selector if multiple groups
	if groupSelector != nil {
		content.Add(widget.NewLabel("Server Group:"))
		content.Add(groupSelector)
	}

	// Add username
	content.Add(widget.NewLabel("Username"))
	content.Add(usernameEntry)

	// Add password
	content.Add(widget.NewLabel("Password"))
	content.Add(passwordEntry)

	// Add OTP if enabled
	if otpContainer != nil {
		content.Add(otpContainer)
	}

	// Add buttons
	if as.Config.Kiosk {
		content.Add(loginButton)
	} else {
		content.Add(container.NewHBox(loginButton, cancelButton))
	}

	as.Window.SetContent(content)
	as.Window.Resize(fyne.NewSize(400, 300))

	// Handle return key
	as.Window.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		if key.Name == fyne.KeyReturn || key.Name == fyne.KeyEnter {
			loginButton.OnTapped()
		}
	})

	as.Window.ShowAndRun()

	return as.LoggedIn
}

// ShowVMWindow displays the VM selection window
func (as *AppState) ShowVMWindow() {
	as.Window = as.App.NewWindow(as.Config.Title)

	// Create VM list
	vmList := container.NewVBox()

	// Get VMs
	vms, err := as.ProxmoxClient.GetVMs()
	if err != nil {
		dialog.ShowError(fmt.Errorf("unable to get VMs: %w", err), as.Window)
		as.Window.Close()
		return
	}

	if len(vms) == 0 {
		dialog.ShowError(fmt.Errorf("no desktop instances found, please consult your system administrator"), as.Window)
		as.Window.Close()
		return
	}

	// Check for auto-connect VM
	hostSet := as.Config.Hosts[as.Config.CurrentHostSet]
	if hostSet.AutoVMID > 0 {
		for _, vm := range vms {
			if vm.VMID == hostSet.AutoVMID {
				err := ConnectToVM(as.ProxmoxClient, vm, as.Config.Kiosk, as.Config.ViewerKiosk, as.Config.Fullscreen)
				if err != nil {
					dialog.ShowError(err, as.Window)
				}
				as.Window.Close()
				return
			}
		}
		dialog.ShowError(fmt.Errorf("no VDI instance with ID %d found", hostSet.AutoVMID), as.Window)
	}

	// Build VM list UI
	for _, vm := range vms {
		vmCopy := vm // Capture for closure

		// VM info
		vmNameLabel := widget.NewLabel(vm.Name)
		vmNameLabel.TextStyle = fyne.TextStyle{Bold: true}

		vmStateLabel := widget.NewLabel(fmt.Sprintf("State: %s", vm.GetDisplayState()))

		// Connect button
		connectBtn := widget.NewButton("Connect", func() {
			progress := dialog.NewCustomWithoutButtons("Connecting", widget.NewLabel(fmt.Sprintf("Connecting to %s...", vmCopy.Name)), as.Window)
			progress.Show()

			go func() {
				err := ConnectToVM(as.ProxmoxClient, vmCopy, as.Config.Kiosk, as.Config.ViewerKiosk, as.Config.Fullscreen)
				progress.Hide()
				if err != nil {
					dialog.ShowError(err, as.Window)
				}
			}()
		})

		if !vm.IsConnectable() {
			connectBtn.Disable()
		}

		// Reset button (if enabled)
		var resetBtn *widget.Button
		if as.Config.ShowReset {
			resetBtn = widget.NewButton("Reset", func() {
				progress := dialog.NewCustomWithoutButtons("Resetting", widget.NewLabel(fmt.Sprintf("Resetting %s...", vmCopy.Name)), as.Window)
				progress.Show()

				go func() {
					err := ResetVM(as.ProxmoxClient, vmCopy)
					progress.Hide()
					if err != nil {
						dialog.ShowError(err, as.Window)
					}
				}()
			})
		}

		// Build VM row
		vmRow := container.NewHBox(vmNameLabel, vmStateLabel, connectBtn)
		if resetBtn != nil {
			vmRow.Add(resetBtn)
		}

		vmList.Add(vmRow)
		vmList.Add(widget.NewSeparator())
	}

	// Logout button
	logoutBtn := widget.NewButton("Logout", func() {
		as.LoggedIn = false
		as.Window.Close()
	})

	// Main content
	content := container.NewBorder(
		container.NewVBox(widget.NewLabel(as.Config.Title)),
		logoutBtn,
		nil,
		nil,
		container.NewScroll(vmList),
	)

	as.Window.SetContent(content)

	// Set window size
	if as.Config.WindowWidth > 0 && as.Config.WindowHeight > 0 {
		as.Window.Resize(fyne.NewSize(float32(as.Config.WindowWidth), float32(as.Config.WindowHeight)))
	} else {
		as.Window.Resize(fyne.NewSize(800, 600))
	}

	// Auto-refresh VM list every 5 seconds
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if as.Window == nil {
				return
			}

			newVMs, err := as.ProxmoxClient.GetVMs()
			if err == nil && len(newVMs) > 0 {
				// Update UI (simplified - in production you'd update existing widgets)
				// For now, we just refresh the whole window
			}
		}
	}()

	as.Window.ShowAndRun()
}

// Run starts the application
func (as *AppState) Run() {
	for {
		if !as.LoggedIn {
			success := as.ShowLoginWindow()
			if !success {
				// User cancelled or auth failed
				return
			}
		}

		// Show VM window
		as.ShowVMWindow()

		// If we get here, user logged out
		as.LoggedIn = false
		as.ProxmoxClient.Client = nil

		// Check if we should exit (token auth mode)
		hostSet := as.Config.Hosts[as.Config.CurrentHostSet]
		if hostSet.User != "" && hostSet.TokenName != "" && hostSet.TokenValue != "" && len(as.Config.Hosts) == 1 {
			return
		}
	}
}
