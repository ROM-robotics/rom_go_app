package main

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rom_go_app/config"
	"rom_go_app/handlers"
	"rom_go_app/robot"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

func main() {
	cfg := config.Load()

	// Parse templates
	tmpl := template.Must(template.ParseFS(templateFS,
		"templates/layout.html",
		"templates/index.html",
		"templates/partials/*.html",
		"templates/dialogs/*.html",
	))

	// Robot manager & navigation manager
	mgr := robot.NewManager()
	nav := robot.NewNavigationManager()

	// Whisper runner (optional)
	whisper := handlers.NewWhisperRunner(cfg.WhisperBinPath, cfg.WhisperModelPath, cfg.SpeechLogDir)

	// Handler server
	srv := &handlers.Server{
		Manager:    mgr,
		NavManager: nav,
		Whisper:    whisper,
		Templates:  tmpl,
	}

	mux := http.NewServeMux()

	// Static files
	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// Pages
	mux.HandleFunc("/", srv.IndexPage)

	// Robot API
	mux.HandleFunc("/api/robots", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			srv.ListRobots(w, r)
		case http.MethodPost:
			srv.AddRobot(w, r)
		case http.MethodDelete:
			srv.RemoveRobot(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/robots/switch", srv.SwitchRobot)
	mux.HandleFunc("/api/robots/status", srv.RobotStatus)
	mux.HandleFunc("/api/robots/velocity_history", srv.GetVelocityHistory)
	mux.HandleFunc("/api/robots/settings", srv.UpdateSettings)
	mux.HandleFunc("/api/robots/task", srv.RequestTask)
	mux.HandleFunc("/api/robots/poweroff", srv.PowerOff)
	mux.HandleFunc("/api/robots/reboot", srv.Reboot)

	// Map API
	mux.HandleFunc("/api/maps", srv.ListMaps)
	mux.HandleFunc("/api/maps/save", srv.SaveMap)
	mux.HandleFunc("/api/maps/open", srv.OpenMap)

	// Mode API
	mux.HandleFunc("/api/mode/navigation", srv.SetNavigationMode)
	mux.HandleFunc("/api/mode/mapping", srv.SetMappingMode)
	mux.HandleFunc("/api/mode/remapping", srv.SetRemappingMode)

	// Navigation API
	mux.HandleFunc("/api/nav/add", srv.AddNavigationPoint)
	mux.HandleFunc("/api/nav/list", srv.ListNavigationPoints)
	mux.HandleFunc("/api/nav/send", srv.SendNavigationPoints)
	mux.HandleFunc("/api/nav/go", srv.GoAllPoints)
	mux.HandleFunc("/api/nav/clear", srv.ClearNavigationPoints)
	mux.HandleFunc("/api/nav/fetch", srv.RequestNavPointsFromRobot)
	mux.HandleFunc("/api/nav/import", srv.ImportNavPoints)
	mux.HandleFunc("/api/nav/delete", srv.DeleteNavPoint)

	// Speech API
	mux.HandleFunc("/api/speech/status", srv.SpeechStatus)
	mux.HandleFunc("/api/speech/transcribe", srv.SpeechTranscribe)

	// HTMX partials
	mux.HandleFunc("/partial/robots", srv.RobotListPartial)
	mux.HandleFunc("/partial/settings", srv.SettingsPartial)
	mux.HandleFunc("/partial/nav_points", srv.NavPointsPartial)

	// Dialog fragments
	mux.HandleFunc("/dialog/add_robot", srv.AddRobotDialog)
	mux.HandleFunc("/dialog/save_map", srv.SaveMapDialog)
	mux.HandleFunc("/dialog/open_map", srv.OpenMapDialog)
	mux.HandleFunc("/dialog/confirm", srv.ConfirmDialog)
	mux.HandleFunc("/dialog/add_nav_point", srv.AddNavPointDialog)

	// WebSocket
	mux.HandleFunc("/ws", srv.WSHandler)

	// HTTP Server
	httpServer := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("[server] Shutting down...")
		mgr.ClearAll()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
	}()

	log.Printf("[server] Listening on %s", cfg.ListenAddr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("[server] Fatal: %v", err)
	}
}
