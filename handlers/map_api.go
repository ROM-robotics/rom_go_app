package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// ListMaps returns available maps from the current robot.
func (s *Server) ListMaps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonError(w, "no robot selected", http.StatusBadRequest)
		return
	}

	maps := rb.GetMapList()

	// If map list is empty, try fetching from robot
	if len(maps) == 0 && rb.Client != nil && rb.Client.IsConnected() {
		names, err := rb.Client.RequestWhichMapsNames()
		if err == nil {
			maps = names
			rb.SetMapList(names)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"maps": maps,
	})
}

// SaveMap saves the current map with a given name.
func (s *Server) SaveMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		jsonError(w, "map name required", http.StatusBadRequest)
		return
	}

	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonError(w, "no robot selected", http.StatusBadRequest)
		return
	}
	if rb.Client == nil || !rb.Client.IsConnected() {
		jsonError(w, "robot not connected", http.StatusServiceUnavailable)
		return
	}

	_, err := rb.Client.SaveMap(req.Name)
	if err != nil {
		log.Printf("[map] save map error: %v", err)
		jsonError(w, "save map failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"status": "ok", "map": req.Name})
}

// OpenMap opens/selects a map by name.
func (s *Server) OpenMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		jsonError(w, "map name required", http.StatusBadRequest)
		return
	}

	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonError(w, "no robot selected", http.StatusBadRequest)
		return
	}
	if rb.Client == nil || !rb.Client.IsConnected() {
		jsonError(w, "robot not connected", http.StatusServiceUnavailable)
		return
	}

	_, err := rb.Client.SelectMap(req.Name)
	if err != nil {
		log.Printf("[map] open map error: %v", err)
		jsonError(w, "open map failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"status": "ok", "map": req.Name})
}

// SetNavigationMode requests navigation mode from the current robot.
func (s *Server) SetNavigationMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonError(w, "no robot selected", http.StatusBadRequest)
		return
	}
	if rb.Client == nil || !rb.Client.IsConnected() {
		jsonError(w, "robot not connected", http.StatusServiceUnavailable)
		return
	}

	_, err := rb.Client.RequestNavigationMode()
	if err != nil {
		jsonError(w, "set navigation mode failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ok", "mode": "navigation"})
}

// SetMappingMode requests mapping mode from the current robot.
func (s *Server) SetMappingMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonError(w, "no robot selected", http.StatusBadRequest)
		return
	}
	if rb.Client == nil || !rb.Client.IsConnected() {
		jsonError(w, "robot not connected", http.StatusServiceUnavailable)
		return
	}

	_, err := rb.Client.RequestMappingMode()
	if err != nil {
		jsonError(w, "set mapping mode failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ok", "mode": "mapping"})
}

// SetRemappingMode requests remapping mode from the current robot.
func (s *Server) SetRemappingMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonError(w, "no robot selected", http.StatusBadRequest)
		return
	}
	if rb.Client == nil || !rb.Client.IsConnected() {
		jsonError(w, "robot not connected", http.StatusServiceUnavailable)
		return
	}

	_, err := rb.Client.RequestRemappingMode()
	if err != nil {
		jsonError(w, "set remapping mode failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ok", "mode": "remapping"})
}

// ──────────────────────────── Dialog handlers

// SaveMapDialog renders the save map dialog fragment.
func (s *Server) SaveMapDialog(w http.ResponseWriter, r *http.Request) {
	s.render(w, "save_map.html", nil)
}

// OpenMapDialog renders the open map dialog fragment.
func (s *Server) OpenMapDialog(w http.ResponseWriter, r *http.Request) {
	// Fetch available maps for the dialog
	var maps []string
	rb := s.Manager.GetCurrentRobot()
	if rb != nil {
		maps = rb.GetMapList()
		// Try refreshing from robot if connected
		if rb.Client != nil && rb.Client.IsConnected() {
			names, err := rb.Client.RequestWhichMapsNames()
			if err == nil && len(names) > 0 {
				maps = names
				rb.SetMapList(names)
			}
		}
	}
	s.render(w, "open_map.html", map[string]interface{}{"Maps": maps})
}

// ConfirmDialog renders a generic confirmation dialog.
func (s *Server) ConfirmDialog(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	message := r.URL.Query().Get("message")
	action := r.URL.Query().Get("action")
	s.render(w, "confirm.html", map[string]interface{}{
		"Title":   title,
		"Message": message,
		"Action":  action,
	})
}
