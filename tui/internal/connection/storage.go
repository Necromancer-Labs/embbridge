package connection

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// SavedDevice represents device data persisted to YAML.
// Contains both user preferences (custom name, color, arch) and
// cached system info for display when device is offline.
type SavedDevice struct {
	// Identifier - used to match reconnecting devices
	ID         string `yaml:"id"`
	RemoteAddr string `yaml:"remote_addr"`

	// User customizations (primary purpose of persistence)
	CustomName  string `yaml:"custom_name,omitempty"`
	CustomArch  string `yaml:"custom_arch,omitempty"`
	CustomColor string `yaml:"custom_color,omitempty"`

	// Cached system info (for display when offline)
	Hostname     string `yaml:"hostname,omitempty"`
	Architecture string `yaml:"architecture,omitempty"`
	OS           string `yaml:"os,omitempty"`
	Kernel       string `yaml:"kernel,omitempty"`

	// Connection history
	LastConnected time.Time `yaml:"last_connected"`
}

// DeviceStore represents the YAML file structure
type DeviceStore struct {
	Devices []SavedDevice `yaml:"devices"`
}

// getStorePath returns the path to the device storage file.
// Uses ~/.graveyard_devices.yaml (consistent with ~/.graveyard_history)
func getStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".graveyard_devices.yaml"), nil
}

// LoadDevices reads saved devices from the YAML file.
// Returns an empty slice if the file doesn't exist (first run).
func LoadDevices() ([]SavedDevice, error) {
	path, err := getStorePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// First run - no saved devices yet
			return []SavedDevice{}, nil
		}
		return nil, err
	}

	var store DeviceStore
	if err := yaml.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	return store.Devices, nil
}

// SaveDevices writes devices to the YAML file.
func SaveDevices(devices []SavedDevice) error {
	path, err := getStorePath()
	if err != nil {
		return err
	}

	store := DeviceStore{Devices: devices}
	data, err := yaml.Marshal(&store)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// DeviceToSaved converts a live Device to SavedDevice for persistence.
func DeviceToSaved(d *Device) SavedDevice {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return SavedDevice{
		ID:            d.ID,
		RemoteAddr:    d.RemoteAddr,
		CustomName:    d.CustomName,
		CustomArch:    d.CustomArch,
		CustomColor:   d.CustomColor,
		Hostname:      d.Hostname,
		Architecture:  d.Architecture,
		OS:            d.OS,
		Kernel:        d.Kernel,
		LastConnected: d.ConnectedAt,
	}
}

// SavedToDevice creates a Device from SavedDevice (for offline display).
// The device starts in Disconnected state.
func SavedToDevice(s SavedDevice) *Device {
	return &Device{
		ID:           s.ID,
		RemoteAddr:   s.RemoteAddr,
		CustomName:   s.CustomName,
		CustomArch:   s.CustomArch,
		CustomColor:  s.CustomColor,
		Hostname:     s.Hostname,
		Architecture: s.Architecture,
		OS:           s.OS,
		Kernel:       s.Kernel,
		ConnectedAt:  s.LastConnected,
		LastActivity: s.LastConnected,
		State:        DeviceDisconnected,
	}
}

// MergeDeviceInfo merges saved preferences into a newly connected device.
// Called when a device reconnects to restore user customizations.
func MergeDeviceInfo(device *Device, saved SavedDevice) {
	device.mu.Lock()
	defer device.mu.Unlock()

	// Restore user customizations
	if saved.CustomName != "" {
		device.CustomName = saved.CustomName
	}
	if saved.CustomArch != "" {
		device.CustomArch = saved.CustomArch
	}
	if saved.CustomColor != "" {
		device.CustomColor = saved.CustomColor
	}
}
