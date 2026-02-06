package robot

import (
	"fmt"
	"log"
	"rom_go_app/rosbridge"
	"sync"
)

// Manager manages the lifecycle of multiple robots.
type Manager struct {
	mu        sync.RWMutex
	robots    map[string]*Robot
	currentID string
	nextID    int

	// Subscriber channels for real-time broadcast
	broadcastMu sync.RWMutex
	subscribers map[chan BroadcastMsg]struct{}
}

// BroadcastMsg is sent to all WebSocket subscribers.
type BroadcastMsg struct {
	Type    string      `json:"type"`
	RobotID string      `json:"robot_id"`
	Data    interface{} `json:"data"`
}

// NewManager creates a new robot manager.
func NewManager() *Manager {
	return &Manager{
		robots:      make(map[string]*Robot),
		nextID:      1,
		subscribers: make(map[chan BroadcastMsg]struct{}),
	}
}

// Subscribe returns a channel for receiving broadcast messages.
func (m *Manager) Subscribe() chan BroadcastMsg {
	ch := make(chan BroadcastMsg, 100)
	m.broadcastMu.Lock()
	m.subscribers[ch] = struct{}{}
	m.broadcastMu.Unlock()
	return ch
}

// Unsubscribe removes a broadcast subscriber.
func (m *Manager) Unsubscribe(ch chan BroadcastMsg) {
	m.broadcastMu.Lock()
	delete(m.subscribers, ch)
	m.broadcastMu.Unlock()
	close(ch)
}

// Broadcast sends a message to all subscribers.
func (m *Manager) Broadcast(msg BroadcastMsg) {
	m.broadcastMu.RLock()
	defer m.broadcastMu.RUnlock()
	for ch := range m.subscribers {
		select {
		case ch <- msg:
		default:
			// Drop if subscriber is slow
		}
	}
}

// AddRobot creates and registers a new robot.
func (m *Manager) AddRobot(ns, name, ip string, port int) (*Robot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check duplicate IP
	for _, r := range m.robots {
		if r.IP == ip && r.Port == port {
			return nil, fmt.Errorf("robot at %s:%d already exists", ip, port)
		}
	}

	id := fmt.Sprintf("%d", m.nextID)
	m.nextID++

	r := NewRobot(id, ns, name, ip, port)

	// Wire up broadcast callbacks for real-time data
	origOnMap := r.Client.OnMap
	r.Client.OnMap = func(md MapData) {
		if origOnMap != nil {
			origOnMap(md)
		}
		m.Broadcast(BroadcastMsg{Type: "map", RobotID: id, Data: md})
	}

	origOnTF := r.Client.OnTF
	r.Client.OnTF = func(tf TFData) {
		if origOnTF != nil {
			origOnTF(tf)
		}
		m.Broadcast(BroadcastMsg{Type: "tf", RobotID: id, Data: tf})
	}

	origOnOdom := r.Client.OnOdom
	r.Client.OnOdom = func(o OdomData) {
		if origOnOdom != nil {
			origOnOdom(o)
		}
		m.Broadcast(BroadcastMsg{Type: "odom", RobotID: id, Data: o})
	}

	origOnCtrlOdom := r.Client.OnCtrlOdom
	r.Client.OnCtrlOdom = func(o OdomData) {
		if origOnCtrlOdom != nil {
			origOnCtrlOdom(o)
		}
		m.Broadcast(BroadcastMsg{Type: "ctrl_odom", RobotID: id, Data: o})
	}

	origOnLaser := r.Client.OnLaser
	r.Client.OnLaser = func(l LaserData) {
		if origOnLaser != nil {
			origOnLaser(l)
		}
		m.Broadcast(BroadcastMsg{Type: "laser", RobotID: id, Data: l})
	}

	origOnTwist := r.Client.OnTwist
	r.Client.OnTwist = func(t TwistData) {
		if origOnTwist != nil {
			origOnTwist(t)
		}
		m.Broadcast(BroadcastMsg{Type: "velocity", RobotID: id, Data: t})
	}

	origOnMapBfp := r.Client.OnMapBfp
	r.Client.OnMapBfp = func(p Pose2D) {
		if origOnMapBfp != nil {
			origOnMapBfp(p)
		}
		m.Broadcast(BroadcastMsg{Type: "map_bfp", RobotID: id, Data: p})
	}

	r.Client.OnConnected = func() {
		r.mu.Lock()
		r.Connected = true
		r.mu.Unlock()
		r.Client.SubscribeAllTopics()
		r.Client.SetCmdVelEnabled(true)
		m.Broadcast(BroadcastMsg{Type: "robot_connected", RobotID: id})
	}

	r.Client.OnDisconnected = func() {
		r.mu.Lock()
		r.Connected = false
		r.mu.Unlock()
		m.Broadcast(BroadcastMsg{Type: "robot_disconnected", RobotID: id})
	}

	m.robots[id] = r

	// Auto-set as current if first
	if m.currentID == "" {
		m.currentID = id
	}

	log.Printf("[manager] Robot added: id=%s name=%s ip=%s:%d", id, name, ip, port)
	m.Broadcast(BroadcastMsg{Type: "robot_added", RobotID: id, Data: r.GetSnapshot()})
	return r, nil
}

// RemoveRobot disconnects and removes a robot.
func (m *Manager) RemoveRobot(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, ok := m.robots[id]
	if !ok {
		return fmt.Errorf("robot %s not found", id)
	}

	r.StopConnection()
	delete(m.robots, id)

	if m.currentID == id {
		m.currentID = ""
		for k := range m.robots {
			m.currentID = k
			break
		}
	}

	m.Broadcast(BroadcastMsg{Type: "robot_removed", RobotID: id})
	log.Printf("[manager] Robot removed: id=%s", id)
	return nil
}

// SwitchRobot sets the current active robot.
func (m *Manager) SwitchRobot(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.robots[id]; !ok {
		return fmt.Errorf("robot %s not found", id)
	}
	m.currentID = id
	m.Broadcast(BroadcastMsg{Type: "robot_switched", RobotID: id})
	return nil
}

// GetCurrentRobot returns the current robot.
func (m *Manager) GetCurrentRobot() *Robot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.currentID == "" {
		return nil
	}
	return m.robots[m.currentID]
}

// GetCurrentRobotID returns the current robot ID.
func (m *Manager) GetCurrentRobotID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentID
}

// GetRobot returns a robot by ID.
func (m *Manager) GetRobot(id string) *Robot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.robots[id]
}

// GetAllRobots returns all robots.
func (m *Manager) GetAllRobots() []*Robot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Robot, 0, len(m.robots))
	for _, r := range m.robots {
		result = append(result, r)
	}
	return result
}

// GetRobotCount returns the number of robots.
func (m *Manager) GetRobotCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.robots)
}

// ClearAll disconnects and removes all robots.
func (m *Manager) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.robots {
		r.StopConnection()
	}
	m.robots = make(map[string]*Robot)
	m.currentID = ""
}

// Type aliases for manager broadcast (avoid import cycles)
type MapData = rosbridge.MapData
type TFData = rosbridge.TFData
type OdomData = rosbridge.OdomData
type LaserData = rosbridge.LaserData
type TwistData = rosbridge.TwistData
type Pose2D = rosbridge.Pose2D
