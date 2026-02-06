package handlers

import (
	"html/template"
	"net/http"

	"rom_go_app/robot"
)

// Server holds shared dependencies for all handlers.
type Server struct {
	Manager    *robot.Manager
	NavManager *robot.NavigationManager
	Whisper    *WhisperRunner
	Templates  *template.Template
}

// IndexPage renders the main application page.
func (s *Server) IndexPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	robots := s.Manager.GetAllRobots()
	data := map[string]interface{}{
		"Robots":    robots,
		"CurrentID": s.Manager.GetCurrentRobotID(),
	}
	s.render(w, "index.html", data)
}

// render executes a template with common error handling.
func (s *Server) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.Templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
