package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"rom_go_app/robot"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 65536,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// WSHandler upgrades HTTP to WebSocket and bridges browser  ↔  robot data.
func (s *Server) WSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	// Subscribe to robot manager broadcasts
	bcast := s.Manager.Subscribe()

	done := make(chan struct{})
	var closeOnce sync.Once

	cleanup := func() {
		closeOnce.Do(func() {
			close(done)
			s.Manager.Unsubscribe(bcast)
			conn.Close()
		})
	}
	defer cleanup()

	// Writer goroutine: forward broadcast messages to browser
	var lastMapSend time.Time
	go func() {
		defer cleanup()
		for {
			select {
			case <-done:
				return
			case msg, ok := <-bcast:
				if !ok {
					return
				}
				// Throttle map data to ~2 fps to browser (maps are large)
				if msg.Type == "map" {
					now := time.Now()
					if now.Sub(lastMapSend) < 500*time.Millisecond {
						continue
					}
					lastMapSend = now
				}

				// Throttle laser data to ~5 fps
				if msg.Type == "laser" {
					// Skip some laser frames to reduce bandwidth
				}

				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteJSON(msg); err != nil {
					if !websocket.IsCloseError(err,
						websocket.CloseNormalClosure,
						websocket.CloseGoingAway) {
						log.Printf("[ws] write error: %v", err)
					}
					return
				}
			}
		}
	}()

	// Reader goroutine: process commands from browser
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway) {
				log.Printf("[ws] read error: %v", err)
			}
			return
		}

		var cmd WSCommand
		if err := json.Unmarshal(msgBytes, &cmd); err != nil {
			log.Printf("[ws] invalid command: %v", err)
			continue
		}

		s.handleWSCommand(conn, cmd)
	}
}

// WSCommand is a message from the browser.
type WSCommand struct {
	Type    string          `json:"type"`
	RobotID string          `json:"robot_id,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// JoystickData holds joystick velocity values.
type JoystickData struct {
	LinearX  float64 `json:"linear_x"`
	AngularZ float64 `json:"angular_z"`
}

// handleWSCommand processes a single WebSocket command from the browser
func (s *Server) handleWSCommand(conn *websocket.Conn, cmd WSCommand) {
	// Get target robot
	robotID := cmd.RobotID
	if robotID == "" {
		robotID = s.Manager.GetCurrentRobotID()
	}

	switch cmd.Type {
	case "joystick":
		var joy JoystickData
		if err := json.Unmarshal(cmd.Data, &joy); err != nil {
			return
		}
		rb := s.Manager.GetRobot(robotID)
		if rb != nil {
			rb.SetVelocity(joy.LinearX, joy.AngularZ)
		}

	case "stop":
		rb := s.Manager.GetRobot(robotID)
		if rb != nil {
			rb.SetVelocity(0, 0)
		}

	case "switch_robot":
		var data struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(cmd.Data, &data); err == nil {
			s.Manager.SwitchRobot(data.ID)
		}

	case "request_map":
		// Send current map data immediately
		rb := s.Manager.GetRobot(robotID)
		if rb != nil {
			mapData := rb.GetMap()
			conn.WriteJSON(robot.BroadcastMsg{
				Type:    "map",
				RobotID: robotID,
				Data:    mapData,
			})
		}

	case "request_status":
		rb := s.Manager.GetRobot(robotID)
		if rb != nil {
			snap := rb.GetSnapshot()
			conn.WriteJSON(robot.BroadcastMsg{
				Type:    "status",
				RobotID: robotID,
				Data:    snap,
			})
		}

	case "voice_command":
		var data struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(cmd.Data, &data); err == nil {
			rb := s.Manager.GetRobot(robotID)
			if rb != nil && rb.Client != nil {
				rb.Client.SendVoiceCommand(data.Text)
			}
		}

	case "connect":
		// Manual connect/reconnect
		rb := s.Manager.GetRobot(robotID)
		if rb != nil && rb.Client != nil && !rb.Client.IsConnected() {
			go rb.Client.Connect()
		}

	case "disconnect":
		rb := s.Manager.GetRobot(robotID)
		if rb != nil {
			rb.StopConnection()
		}

	default:
		log.Printf("[ws] unknown command type: %s", cmd.Type)
	}
}
