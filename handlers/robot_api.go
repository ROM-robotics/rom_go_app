package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// ──────────────────── Robot CRUD ────────────────────

// AddRobot handles POST /api/robots
func (s *Server) AddRobot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ns := r.FormValue("namespace")
	name := r.FormValue("name")
	ip := r.FormValue("ip")
	portStr := r.FormValue("port")

	if ns == "" || name == "" || ip == "" {
		jsonError(w, "namespace, name, and ip are required", http.StatusBadRequest)
		return
	}

	port := 9090
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			jsonError(w, "invalid port", http.StatusBadRequest)
			return
		}
		port = p
	}

	robot, err := s.Manager.AddRobot(ns, name, ip, port)
	if err != nil {
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}

	// Start connection in background
	go func() {
		if err := robot.Client.Connect(); err != nil {
			log.Printf("[api] Robot connect error: %v", err)
			return
		}
		// Handshake to get robot info
		hs, err := robot.Client.Handshake()
		if err != nil {
			log.Printf("[api] Handshake failed for %s: %v", name, err)
		} else {
			log.Printf("[api] Handshake OK: ns=%s diameter=%.2f", hs.RobotNamespace, hs.RobotDiameter)
			if hs.RobotDiameter > 0 {
				robot.SetRadius(hs.RobotDiameter / 2.0)
			}
		}
	}()

	log.Printf("[api] Robot added: %s (%s:%d)", name, ip, port)

	// If HTMX request, return the updated robot list partial
	if r.Header.Get("HX-Request") == "true" {
		s.RobotListPartial(w, r)
		return
	}

	jsonOK(w, map[string]interface{}{
		"id":   robot.ID,
		"name": robot.Name,
		"ip":   robot.IP,
	})
}

// RemoveRobot handles DELETE /api/robots?id=X
func (s *Server) RemoveRobot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		jsonError(w, "id required", http.StatusBadRequest)
		return
	}

	if err := s.Manager.RemoveRobot(id); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		s.RobotListPartial(w, r)
		return
	}

	jsonOK(w, map[string]string{"status": "removed"})
}

// SwitchRobot handles POST /api/robots/switch?id=X
func (s *Server) SwitchRobot(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if err := s.Manager.SwitchRobot(id); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		s.RobotListPartial(w, r)
		return
	}

	jsonOK(w, map[string]string{"status": "switched", "id": id})
}

// ListRobots handles GET /api/robots
func (s *Server) ListRobots(w http.ResponseWriter, r *http.Request) {
	robots := s.Manager.GetAllRobots()
	list := make([]map[string]interface{}, 0, len(robots))
	currentID := s.Manager.GetCurrentRobotID()

	for _, rb := range robots {
		snap := rb.GetSnapshot()
		list = append(list, map[string]interface{}{
			"id":        snap.ID,
			"namespace": snap.Namespace,
			"name":      snap.Name,
			"ip":        snap.IP,
			"port":      snap.Port,
			"connected": snap.Connected,
			"current":   snap.ID == currentID,
		})
	}

	jsonOK(w, list)
}

// RobotStatus handles GET /api/robots/status?id=X
func (s *Server) RobotStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = s.Manager.GetCurrentRobotID()
	}

	rb := s.Manager.GetRobot(id)
	if rb == nil {
		jsonError(w, "robot not found", http.StatusNotFound)
		return
	}

	snap := rb.GetSnapshot()
	jsonOK(w, map[string]interface{}{
		"id":        snap.ID,
		"name":      snap.Name,
		"connected": snap.Connected,
		"odom":      snap.Odom,
		"velocity":  snap.Velocity,
		"map_hz":    snap.MapHz,
		"tf_hz":     snap.TFHz,
		"odom_hz":   snap.OdomHz,
		"laser_hz":  snap.LaserHz,
	})
}

// GetVelocityHistory handles GET /api/robots/velocity_history?id=X
func (s *Server) GetVelocityHistory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = s.Manager.GetCurrentRobotID()
	}

	rb := s.Manager.GetRobot(id)
	if rb == nil {
		jsonError(w, "robot not found", http.StatusNotFound)
		return
	}

	hist := rb.GetVelocityHistory()
	jsonOK(w, hist)
}

// UpdateSettings handles POST /api/robots/settings
func (s *Server) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = s.Manager.GetCurrentRobotID()
	}

	rb := s.Manager.GetRobot(id)
	if rb == nil {
		jsonError(w, "robot not found", http.StatusNotFound)
		return
	}

	if v := r.FormValue("linear_vel_ratio"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			rb.LinearVelRatio = f
		}
	}
	if v := r.FormValue("angular_vel_ratio"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			rb.AngularVelRatio = f
		}
	}
	if v := r.FormValue("radius"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			rb.Radius = f
		}
	}

	// Send settings to robot if connected
	if rb.Connected && rb.Client != nil {
		args := map[string]interface{}{
			"linear_vel_ratio":  rb.LinearVelRatio,
			"angular_vel_ratio": rb.AngularVelRatio,
			"radius":            rb.Radius,
		}
		argsJSON, _ := json.Marshal(args)
		_, _ = rb.Client.RequestSettingsSave(string(argsJSON))
	}

	jsonOK(w, map[string]string{"status": "updated"})
}

// ──────────────────── Task commands ────────────────────

// RequestTask handles POST /api/robots/task
func (s *Server) RequestTask(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = s.Manager.GetCurrentRobotID()
	}
	task := r.FormValue("task")

	rb := s.Manager.GetRobot(id)
	if rb == nil || rb.Client == nil {
		jsonError(w, "robot not found", http.StatusNotFound)
		return
	}

	settings := r.FormValue("settings")
	resp, err := rb.Client.RequestTask(task, settings)
	if err != nil {
		jsonError(w, fmt.Sprintf("task '%s' failed: %v", task, err), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]interface{}{"result": resp})
}

// PowerOff handles POST /api/robots/poweroff
func (s *Server) PowerOff(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = s.Manager.GetCurrentRobotID()
	}

	rb := s.Manager.GetRobot(id)
	if rb == nil || rb.Client == nil {
		jsonError(w, "robot not found", http.StatusNotFound)
		return
	}

	_, err := rb.Client.RequestPowerOff()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"status": "power_off_sent"})
}

// Reboot handles POST /api/robots/reboot
func (s *Server) Reboot(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		id = s.Manager.GetCurrentRobotID()
	}

	rb := s.Manager.GetRobot(id)
	if rb == nil || rb.Client == nil {
		jsonError(w, "robot not found", http.StatusNotFound)
		return
	}

	_, err := rb.Client.RequestReboot()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"status": "reboot_sent"})
}

// ──────────────────── HTMX Partials ────────────────────

// RobotListPartial renders the robot list for HTMX swap.
func (s *Server) RobotListPartial(w http.ResponseWriter, r *http.Request) {
	robots := s.Manager.GetAllRobots()
	data := map[string]interface{}{
		"Robots":    robots,
		"CurrentID": s.Manager.GetCurrentRobotID(),
	}
	s.render(w, "robot_panel.html", data)
}

// AddRobotDialog renders the add-robot dialog fragment.
func (s *Server) AddRobotDialog(w http.ResponseWriter, r *http.Request) {
	s.render(w, "add_robot.html", nil)
}

// SettingsPartial renders the settings panel.
func (s *Server) SettingsPartial(w http.ResponseWriter, r *http.Request) {
	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		s.render(w, "settings_panel.html", nil)
		return
	}
	snap := rb.GetSnapshot()
	s.render(w, "settings_panel.html", snap)
}

// ──────────────────── Helpers ────────────────────

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
