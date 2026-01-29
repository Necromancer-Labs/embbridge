package connection

import (
	"sync"
	"time"
)

// DeviceState represents the current state of a device connection
type DeviceState int

const (
	DeviceConnecting DeviceState = iota
	DeviceConnected
	DeviceBusy
	DeviceTransferring
	DeviceError
	DeviceDisconnected
)

// String returns a human-readable state name
func (s DeviceState) String() string {
	switch s {
	case DeviceConnecting:
		return "Connecting"
	case DeviceConnected:
		return "Connected"
	case DeviceBusy:
		return "Busy"
	case DeviceTransferring:
		return "Transferring"
	case DeviceError:
		return "Error"
	case DeviceDisconnected:
		return "Disconnected"
	default:
		return "Unknown"
	}
}

// Device represents a connected embedded device
type Device struct {
	ID           string
	RemoteAddr   string
	ConnectedAt  time.Time
	LastActivity time.Time
	State        DeviceState

	// Custom name set by user (overrides Hostname in display)
	CustomName string

	// Custom architecture set by user (overrides detected arch)
	CustomArch string

	// Custom highlight color for this device (hex color like "#ff0000")
	CustomColor string

	// Device info (populated from uname/whoami)
	Hostname     string
	Kernel       string
	Architecture string
	OS           string
	User         string
	UID          int
	GID          int

	// Current state
	CurrentDir string

	// Error tracking
	LastError  error
	ErrorCount int

	// Transfer state
	ActiveTransfer *TransferInfo

	// Internal
	mu      sync.RWMutex
	session *Session
	manager *Manager // Reference to manager for save callbacks
}

// TransferType indicates pull or push
type TransferType int

const (
	TransferPull TransferType = iota
	TransferPush
)

// TransferInfo tracks an active file transfer
type TransferInfo struct {
	Type        TransferType
	RemotePath  string
	LocalPath   string
	TotalBytes  int64
	Transferred int64
	StartTime   time.Time
}

// NewDevice creates a new device with the given ID and address
func NewDevice(id, remoteAddr string) *Device {
	return &Device{
		ID:           id,
		RemoteAddr:   remoteAddr,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		State:        DeviceConnecting,
		CurrentDir:   "/",
	}
}

// SetState safely sets the device state
func (d *Device) SetState(state DeviceState) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.State = state
	d.LastActivity = time.Now()
}

// GetState safely gets the device state
func (d *Device) GetState() DeviceState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.State
}

// SetSession sets the session for this device
func (d *Device) SetSession(s *Session) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.session = s
}

// GetSession returns the session for this device
func (d *Device) GetSession() *Session {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.session
}

// SetError records an error for this device
func (d *Device) SetError(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.LastError = err
	d.ErrorCount++
	d.State = DeviceError
}

// UpdateInfo updates device info from uname response
func (d *Device) UpdateInfo(hostname, kernel, arch, os string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Hostname = hostname
	d.Kernel = kernel
	d.Architecture = arch
	d.OS = os
}

// UpdateUser updates user info from whoami response
func (d *Device) UpdateUser(user string, uid, gid int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.User = user
	d.UID = uid
	d.GID = gid
}

// SetCurrentDir sets the current directory
func (d *Device) SetCurrentDir(dir string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.CurrentDir = dir
}

// GetCurrentDir returns the current directory
func (d *Device) GetCurrentDir() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CurrentDir
}

// DisplayName returns the best name to display for this device.
// Priority: CustomName (user-set) > Hostname (from device) > RemoteAddr (fallback)
func (d *Device) DisplayName() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.CustomName != "" {
		return d.CustomName
	}
	if d.Hostname != "" {
		return d.Hostname
	}
	return d.RemoteAddr
}

// SetCustomName sets a user-defined name for this device.
// This overrides the hostname in DisplayName().
func (d *Device) SetCustomName(name string) {
	d.mu.Lock()
	d.CustomName = name
	manager := d.manager
	d.mu.Unlock()

	// Trigger save
	if manager != nil {
		manager.SaveAllDevices()
	}
}

// GetCustomName returns the user-defined name, or empty string if not set.
func (d *Device) GetCustomName() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CustomName
}

// DisplayArch returns the best architecture to display for this device.
// Priority: CustomArch (user-set) > Architecture (from device) > "?" (fallback)
func (d *Device) DisplayArch() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.CustomArch != "" {
		return d.CustomArch
	}
	if d.Architecture != "" {
		return d.Architecture
	}
	return "?"
}

// SetCustomArch sets a user-defined architecture for this device.
// This overrides the detected architecture in display.
func (d *Device) SetCustomArch(arch string) {
	d.mu.Lock()
	d.CustomArch = arch
	manager := d.manager
	d.mu.Unlock()

	// Trigger save
	if manager != nil {
		manager.SaveAllDevices()
	}
}

// GetCustomArch returns the user-defined architecture, or empty string if not set.
func (d *Device) GetCustomArch() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CustomArch
}

// SetCustomColor sets a custom highlight color for this device.
// Use hex format like "#ff0000" for red, or empty string to clear.
func (d *Device) SetCustomColor(color string) {
	d.mu.Lock()
	d.CustomColor = color
	manager := d.manager
	d.mu.Unlock()

	// Trigger save
	if manager != nil {
		manager.SaveAllDevices()
	}
}

// GetCustomColor returns the custom highlight color, or empty string if not set.
func (d *Device) GetCustomColor() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CustomColor
}

// ConnectionDuration returns how long the device has been connected
func (d *Device) ConnectionDuration() time.Duration {
	return time.Since(d.ConnectedAt)
}

// StartTransfer starts tracking a file transfer
func (d *Device) StartTransfer(transferType TransferType, remotePath, localPath string, totalBytes int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveTransfer = &TransferInfo{
		Type:       transferType,
		RemotePath: remotePath,
		LocalPath:  localPath,
		TotalBytes: totalBytes,
		StartTime:  time.Now(),
	}
	d.State = DeviceTransferring
}

// UpdateTransferProgress updates transfer progress
func (d *Device) UpdateTransferProgress(transferred int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ActiveTransfer != nil {
		d.ActiveTransfer.Transferred = transferred
	}
}

// EndTransfer ends the current transfer
func (d *Device) EndTransfer() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveTransfer = nil
	d.State = DeviceConnected
}
