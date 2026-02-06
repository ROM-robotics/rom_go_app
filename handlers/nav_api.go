package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"rom_go_app/rosbridge"
)

// ──────────────────── Navigation Point API ────────────────────

// AddNavigationPoint handles POST /api/nav/add
func (s *Server) AddNavigationPoint(w http.ResponseWriter, r *http.Request) {
	pointType := r.FormValue("type") // waypoint, service_point, patrol_point, path_point, wall
	name := r.FormValue("name")
	xStr := r.FormValue("world_x")
	yStr := r.FormValue("world_y")
	thetaStr := r.FormValue("theta")

	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonError(w, "no active robot", http.StatusBadRequest)
		return
	}

	x, _ := strconv.ParseFloat(xStr, 64)
	y, _ := strconv.ParseFloat(yStr, 64)
	theta, _ := strconv.ParseFloat(thetaStr, 64)

	var err error
	switch pointType {
	case "waypoint":
		err = s.NavManager.AddWaypoint(rb, name, x, y, theta)
	case "service_point":
		err = s.NavManager.AddServicePoint(rb, name, x, y, theta)
	case "patrol_point":
		err = s.NavManager.AddPatrolPoint(rb, name, x, y, theta)
	case "path_point":
		err = s.NavManager.AddPathPoint(rb, name, x, y, theta)
	case "wall":
		x2, _ := strconv.ParseFloat(r.FormValue("world_x2"), 64)
		y2, _ := strconv.ParseFloat(r.FormValue("world_y2"), 64)
		err = s.NavManager.AddWallObstacle(rb, name, x, y, x2, y2)
	default:
		jsonError(w, "invalid point type", http.StatusBadRequest)
		return
	}

	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		s.NavPointsPartial(w, r)
		return
	}

	jsonOK(w, map[string]string{"status": "added"})
}

// ListNavigationPoints handles GET /api/nav/list?type=X
func (s *Server) ListNavigationPoints(w http.ResponseWriter, r *http.Request) {
	pointType := r.URL.Query().Get("type")

	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonOK(w, []interface{}{})
		return
	}

	snap := rb.GetSnapshot()

	var points interface{}
	switch pointType {
	case "waypoint":
		points = snap.Waypoints
	case "service_point":
		points = snap.ServicePoints
	case "patrol_point":
		points = snap.PatrolPoints
	case "path_point":
		points = snap.PathPoints
	case "wall":
		points = snap.WallObstacles
	default:
		points = map[string]interface{}{
			"waypoints":      snap.Waypoints,
			"service_points": snap.ServicePoints,
			"patrol_points":  snap.PatrolPoints,
			"path_points":    snap.PathPoints,
			"wall_obstacles": snap.WallObstacles,
		}
	}

	jsonOK(w, points)
}

// SendNavigationPoints handles POST /api/nav/send?type=X
func (s *Server) SendNavigationPoints(w http.ResponseWriter, r *http.Request) {
	pointType := r.FormValue("type")

	rb := s.Manager.GetCurrentRobot()
	if rb == nil || rb.Client == nil {
		jsonError(w, "no active robot", http.StatusBadRequest)
		return
	}

	var err error
	switch pointType {
	case "waypoint":
		err = s.NavManager.SendWaypointsToRobot(rb)
	case "service_point":
		err = s.NavManager.SendServicePointsToRobot(rb)
	case "patrol_point":
		err = s.NavManager.SendPatrolPointsToRobot(rb)
	case "path_point":
		err = s.NavManager.SendPathPointsToRobot(rb)
	case "wall":
		err = s.NavManager.SendWallObstaclesToRobot(rb)
	default:
		jsonError(w, "invalid point type", http.StatusBadRequest)
		return
	}

	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"status": "sent"})
}

// GoAllPoints handles POST /api/nav/go?type=X
func (s *Server) GoAllPoints(w http.ResponseWriter, r *http.Request) {
	pointType := r.FormValue("type")

	rb := s.Manager.GetCurrentRobot()
	if rb == nil || rb.Client == nil {
		jsonError(w, "no active robot", http.StatusBadRequest)
		return
	}

	var err error
	switch pointType {
	case "waypoint":
		err = s.NavManager.GoAllWaypoints(rb)
	case "service_point":
		err = s.NavManager.GoAllServicePoints(rb)
	case "patrol_point":
		err = s.NavManager.GoAllPatrolPoints(rb)
	case "path_point":
		err = s.NavManager.GoAllPathPoints(rb)
	default:
		jsonError(w, "invalid point type", http.StatusBadRequest)
		return
	}

	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"status": "go_all_sent"})
}

// ClearNavigationPoints handles POST /api/nav/clear?type=X
func (s *Server) ClearNavigationPoints(w http.ResponseWriter, r *http.Request) {
	pointType := r.FormValue("type")

	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonError(w, "no active robot", http.StatusBadRequest)
		return
	}

	switch pointType {
	case "waypoint":
		s.NavManager.ClearWaypoints(rb)
	case "service_point":
		s.NavManager.ClearServicePoints(rb)
	case "patrol_point":
		s.NavManager.ClearPatrolPoints(rb)
	case "path_point":
		s.NavManager.ClearPathPoints(rb)
	case "wall":
		_ = s.NavManager.ClearWallObstacles(rb)
	default:
		jsonError(w, "invalid point type", http.StatusBadRequest)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		s.NavPointsPartial(w, r)
		return
	}

	jsonOK(w, map[string]string{"status": "cleared"})
}

// RequestNavPointsFromRobot handles POST /api/nav/fetch?type=X
func (s *Server) RequestNavPointsFromRobot(w http.ResponseWriter, r *http.Request) {
	pointType := r.FormValue("type")

	rb := s.Manager.GetCurrentRobot()
	if rb == nil || rb.Client == nil {
		jsonError(w, "no active robot", http.StatusBadRequest)
		return
	}

	var err error
	switch pointType {
	case "waypoint":
		err = s.NavManager.RequestWaypoints(rb)
	case "service_point":
		err = s.NavManager.RequestServicePoints(rb)
	case "patrol_point":
		err = s.NavManager.RequestPatrolPoints(rb)
	case "path_point":
		err = s.NavManager.RequestPathPoints(rb)
	default:
		jsonError(w, "invalid point type", http.StatusBadRequest)
		return
	}

	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "fetching"})
}

// ImportNavPoints handles POST /api/nav/import (JSON upload)
func (s *Server) ImportNavPoints(w http.ResponseWriter, r *http.Request) {
	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonError(w, "no active robot", http.StatusBadRequest)
		return
	}

	var payload struct {
		Type   string                      `json:"type"`
		Points []rosbridge.NavigationPoint `json:"points"`
		Walls  []rosbridge.WallObstacle    `json:"walls,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	rb.ImportPoints(payload.Type, payload.Points, payload.Walls)

	jsonOK(w, map[string]string{"status": "imported"})
}

// NavPointsPartial renders the navigation points panel for HTMX.
func (s *Server) NavPointsPartial(w http.ResponseWriter, r *http.Request) {
	rb := s.Manager.GetCurrentRobot()
	data := map[string]interface{}{}
	if rb != nil {
		wp, sp, pp, pathP, walls := s.NavManager.GetCounts(rb)
		data["Counts"] = map[string]int{
			"waypoints":      wp,
			"service_points": sp,
			"patrol_points":  pp,
			"path_points":    pathP,
			"wall_obstacles": walls,
		}
		snap := rb.GetSnapshot()
		data["Waypoints"] = snap.Waypoints
		data["ServicePoints"] = snap.ServicePoints
		data["PatrolPoints"] = snap.PatrolPoints
		data["PathPoints"] = snap.PathPoints
		data["WallObstacles"] = snap.WallObstacles
	}
	s.render(w, "nav_points.html", data)
}

// AddNavPointDialog renders the add-nav-point dialog for HTMX.
func (s *Server) AddNavPointDialog(w http.ResponseWriter, r *http.Request) {
	pointType := r.URL.Query().Get("type")
	if pointType == "" {
		pointType = "waypoint"
	}
	s.render(w, "add_nav_point.html", map[string]interface{}{
		"Type": pointType,
	})
}

// DeleteNavPoint handles DELETE /api/nav/delete?type=X&name=Y
func (s *Server) DeleteNavPoint(w http.ResponseWriter, r *http.Request) {
	pointType := r.URL.Query().Get("type")
	name := r.URL.Query().Get("name")

	rb := s.Manager.GetCurrentRobot()
	if rb == nil {
		jsonError(w, "no active robot", http.StatusBadRequest)
		return
	}

	s.NavManager.DeletePoint(rb, pointType, name)

	if r.Header.Get("HX-Request") == "true" {
		s.NavPointsPartial(w, r)
		return
	}

	jsonOK(w, map[string]string{"status": "deleted"})
}
