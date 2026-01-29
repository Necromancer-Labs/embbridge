package connection

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DeviceEventType represents the type of device event
type DeviceEventType int

const (
	EventDeviceAdded DeviceEventType = iota
	EventDeviceRemoved
	EventDeviceUpdated
)

// DeviceEvent is sent when device state changes
type DeviceEvent struct {
	Type   DeviceEventType
	Device *Device
	ID     string // For removal events
}

// Manager handles device connections
type Manager struct {
	listener net.Listener
	devices  map[string]*Device
	mu       sync.RWMutex

	// Channel for sending events to the TUI
	events chan DeviceEvent

	// Configuration
	listenAddr string

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Saved device data (for matching reconnections)
	savedDevices map[string]SavedDevice // keyed by RemoteAddr
}

// NewManager creates a new connection manager.
// Loads previously saved devices from YAML and displays them as offline.
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		devices:      make(map[string]*Device),
		savedDevices: make(map[string]SavedDevice),
		events:       make(chan DeviceEvent, 100),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Load saved devices from YAML
	saved, err := LoadDevices()
	if err == nil {
		for _, s := range saved {
			// Store for matching reconnections
			m.savedDevices[s.RemoteAddr] = s
			// Create offline device entries for display
			device := SavedToDevice(s)
			device.manager = m // Set manager reference for save callbacks
			m.devices[device.ID] = device
		}
	}

	return m
}

// Events returns the channel for receiving device events
func (m *Manager) Events() <-chan DeviceEvent {
	return m.events
}

// Listen starts accepting connections on the given address
func (m *Manager) Listen(addr string) error {
	m.listenAddr = addr

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	m.listener = listener

	// Start accept loop in background
	go m.acceptLoop()

	return nil
}

// ListenAddr returns the address the manager is listening on
func (m *Manager) ListenAddr() string {
	if m.listener == nil {
		return ""
	}
	return m.listener.Addr().String()
}

// acceptLoop accepts incoming connections
func (m *Manager) acceptLoop() {
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
		}

		conn, err := m.listener.Accept()
		if err != nil {
			select {
			case <-m.ctx.Done():
				return
			default:
				continue
			}
		}

		// Handle connection in background
		go m.handleIncomingConnection(conn)
	}
}

// handleIncomingConnection handles a new incoming agent connection (reverse mode)
func (m *Manager) handleIncomingConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()

	// Check if we have a saved device for this address (reconnection)
	var device *Device
	m.mu.Lock()
	if saved, ok := m.savedDevices[remoteAddr]; ok {
		// Find existing device entry
		for _, d := range m.devices {
			if d.RemoteAddr == remoteAddr {
				device = d
				break
			}
		}
		if device != nil {
			// Reconnecting device - restore it
			device.mu.Lock()
			device.ConnectedAt = time.Now()
			device.LastActivity = time.Now()
			device.State = DeviceConnecting
			device.mu.Unlock()
			// Merge saved preferences
			MergeDeviceInfo(device, saved)
		}
	}
	m.mu.Unlock()

	// Create new device if not a reconnection
	if device == nil {
		id := uuid.New().String()[:8]
		device = NewDevice(id, remoteAddr)
	}

	// Create session
	session := NewSession(conn, device)
	device.SetSession(session)

	// Perform handshake (we're the server, agent sends hello first)
	if err := session.PerformHandshakeAsServer(); err != nil {
		conn.Close()
		return
	}

	// Initialize device info
	if err := session.Initialize(); err != nil {
		device.SetError(err)
		conn.Close()
		return
	}

	// Add to tracking (handles both new and reconnected devices)
	m.addDevice(device)

	// Start monitoring
	go m.monitorDevice(device)
}

// Connect connects to an agent in bind mode
func (m *Manager) Connect(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", addr, err)
	}

	// Create device
	id := uuid.New().String()[:8]
	device := NewDevice(id, addr)

	// Create session
	session := NewSession(conn, device)
	device.SetSession(session)

	// Perform handshake (we're the client, we send hello first)
	if err := session.PerformHandshakeAsClient(); err != nil {
		conn.Close()
		return fmt.Errorf("handshake failed: %w", err)
	}

	// Initialize device info
	if err := session.Initialize(); err != nil {
		conn.Close()
		return fmt.Errorf("initialize failed: %w", err)
	}

	// Add to tracking
	m.addDevice(device)

	// Start monitoring
	go m.monitorDevice(device)

	return nil
}

// addDevice adds a device to tracking
func (m *Manager) addDevice(device *Device) {
	m.mu.Lock()
	device.manager = m // Set manager reference for save callbacks
	m.devices[device.ID] = device
	// Update savedDevices mapping for reconnection matching
	m.savedDevices[device.RemoteAddr] = DeviceToSaved(device)
	m.mu.Unlock()

	// Notify TUI
	select {
	case m.events <- DeviceEvent{Type: EventDeviceAdded, Device: device}:
	default:
	}

	// Save to YAML
	m.SaveAllDevices()
}

// removeDevice marks a device as disconnected (keeps it for persistence).
// The device remains in the list as "DEAD" so user can reconnect later.
func (m *Manager) removeDevice(id string) {
	m.mu.Lock()
	device, ok := m.devices[id]
	m.mu.Unlock()

	if !ok {
		return
	}

	// Mark as disconnected instead of removing
	device.SetState(DeviceDisconnected)

	// Update saved data
	m.mu.Lock()
	m.savedDevices[device.RemoteAddr] = DeviceToSaved(device)
	m.mu.Unlock()

	// Notify TUI of state change
	select {
	case m.events <- DeviceEvent{Type: EventDeviceUpdated, Device: device}:
	default:
	}

	// Save to YAML
	m.SaveAllDevices()
}

// monitorDevice monitors a device for disconnection
func (m *Manager) monitorDevice(device *Device) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	session := device.GetSession()
	if session == nil {
		return
	}

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-session.Context().Done():
			device.SetState(DeviceDisconnected)
			m.removeDevice(device.ID)
			return
		case <-ticker.C:
			// Heartbeat - send a simple command to check connection
			if _, err := session.Pwd(); err != nil {
				device.SetState(DeviceDisconnected)
				device.SetError(err)
				session.Close()
				m.removeDevice(device.ID)
				return
			}
		}
	}
}

// Devices returns a snapshot of connected devices
func (m *Manager) Devices() []*Device {
	m.mu.RLock()
	defer m.mu.RUnlock()

	devices := make([]*Device, 0, len(m.devices))
	for _, d := range m.devices {
		devices = append(devices, d)
	}
	return devices
}

// DeviceCount returns the number of connected devices
func (m *Manager) DeviceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.devices)
}

// GetDevice returns a device by ID
func (m *Manager) GetDevice(id string) (*Device, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.devices[id]
	return d, ok
}

// Disconnect disconnects a device (keeps it in list as DEAD)
func (m *Manager) Disconnect(id string) error {
	m.mu.RLock()
	device, ok := m.devices[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("device not found: %s", id)
	}

	session := device.GetSession()
	if session != nil {
		session.Close()
	}

	// removeDevice now marks as disconnected instead of deleting
	m.removeDevice(id)
	return nil
}

// Stop stops the manager and closes all connections
func (m *Manager) Stop() error {
	m.cancel()

	if m.listener != nil {
		m.listener.Close()
	}

	// Save all devices before shutdown
	m.SaveAllDevices()

	// Close all device sessions
	m.mu.Lock()
	for _, device := range m.devices {
		if session := device.GetSession(); session != nil {
			session.Close()
		}
	}
	m.devices = make(map[string]*Device)
	m.mu.Unlock()

	close(m.events)
	return nil
}

// SaveAllDevices persists all devices to YAML.
// Called on customization changes, disconnections, and shutdown.
func (m *Manager) SaveAllDevices() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var saved []SavedDevice
	for _, device := range m.devices {
		saved = append(saved, DeviceToSaved(device))
	}

	// Fire and forget - don't block on save errors
	go SaveDevices(saved)
}

// RemoveDevicePermanently removes a device from tracking and from saved storage.
// Used when user explicitly wants to forget a device.
func (m *Manager) RemoveDevicePermanently(id string) {
	m.mu.Lock()
	device, ok := m.devices[id]
	if ok {
		// Remove from saved devices mapping
		delete(m.savedDevices, device.RemoteAddr)
		delete(m.devices, id)
	}
	m.mu.Unlock()

	if ok {
		// Notify TUI
		select {
		case m.events <- DeviceEvent{Type: EventDeviceRemoved, ID: id}:
		default:
		}
		// Save updated list
		m.SaveAllDevices()
	}
}

// ReconnectDevice attempts to reconnect to a disconnected device.
// Returns an error if the device is not found or already connected.
func (m *Manager) ReconnectDevice(id string) error {
	m.mu.RLock()
	device, ok := m.devices[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("device not found: %s", id)
	}

	if device.GetState() != DeviceDisconnected {
		return fmt.Errorf("device is not disconnected")
	}

	// Connect to the device's address
	return m.connectToExisting(device)
}

// connectToExisting connects to an existing device entry (reconnection).
func (m *Manager) connectToExisting(existingDevice *Device) error {
	conn, err := net.DialTimeout("tcp", existingDevice.RemoteAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", existingDevice.RemoteAddr, err)
	}

	// Create session
	session := NewSession(conn, existingDevice)
	existingDevice.SetSession(session)

	// Perform handshake (we're the client, we send hello first)
	if err := session.PerformHandshakeAsClient(); err != nil {
		conn.Close()
		return fmt.Errorf("handshake failed: %w", err)
	}

	// Initialize device info
	if err := session.Initialize(); err != nil {
		conn.Close()
		return fmt.Errorf("initialize failed: %w", err)
	}

	// Update timestamps
	existingDevice.mu.Lock()
	existingDevice.ConnectedAt = time.Now()
	existingDevice.LastActivity = time.Now()
	existingDevice.State = DeviceConnected
	existingDevice.manager = m
	existingDevice.mu.Unlock()

	// Notify TUI
	select {
	case m.events <- DeviceEvent{Type: EventDeviceUpdated, Device: existingDevice}:
	default:
	}

	// Start monitoring
	go m.monitorDevice(existingDevice)

	// Save updated connection time
	m.SaveAllDevices()

	return nil
}
