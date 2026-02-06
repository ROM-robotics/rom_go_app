package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WhisperRunner handles speech-to-text via whisper.cpp CLI.
type WhisperRunner struct {
	BinPath   string
	ModelPath string
	LogDir    string
}

// NewWhisperRunner creates a WhisperRunner if paths exist.
func NewWhisperRunner(binPath, modelPath, logDir string) *WhisperRunner {
	return &WhisperRunner{
		BinPath:   binPath,
		ModelPath: modelPath,
		LogDir:    logDir,
	}
}

// Ready returns true if whisper binary and model exist.
func (wr *WhisperRunner) Ready() bool {
	if wr == nil {
		return false
	}
	if _, err := os.Stat(wr.BinPath); err != nil {
		return false
	}
	if _, err := os.Stat(wr.ModelPath); err != nil {
		return false
	}
	return true
}

// Transcribe converts an audio file to text using whisper.cpp.
func (wr *WhisperRunner) Transcribe(audioPath string) (string, error) {
	if !wr.Ready() {
		return "", fmt.Errorf("whisper not available")
	}

	// Convert to WAV 16kHz mono using ffmpeg
	wavPath := strings.TrimSuffix(audioPath, filepath.Ext(audioPath)) + "_16k.wav"
	ffmpegCmd := exec.Command("ffmpeg", "-y", "-i", audioPath, "-ar", "16000", "-ac", "1", "-f", "wav", wavPath)
	if out, err := ffmpegCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w: %s", err, string(out))
	}
	defer os.Remove(wavPath)

	// Run whisper.cpp
	whisperCmd := exec.Command(wr.BinPath, "-m", wr.ModelPath, "-f", wavPath, "--no-timestamps", "-nt")
	out, err := whisperCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("whisper failed: %w: %s", err, string(out))
	}

	text := strings.TrimSpace(string(out))
	return text, nil
}

// ──────────────────────────── HTTP Handlers

// SpeechStatus returns whether whisper is available.
func (s *Server) SpeechStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ready := s.Whisper != nil && s.Whisper.Ready()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"available": ready,
	})
}

// SpeechTranscribe receives audio, transcribes it, and optionally sends as voice command.
func (s *Server) SpeechTranscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.Whisper == nil || !s.Whisper.Ready() {
		jsonError(w, "whisper not available", http.StatusServiceUnavailable)
		return
	}

	// Parse multipart form (max 10 MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, "invalid form data: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		jsonError(w, "audio file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save uploaded audio to log directory
	os.MkdirAll(s.Whisper.LogDir, 0755)
	ts := time.Now().Format("20060102_150405")
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".webm"
	}
	audioPath := filepath.Join(s.Whisper.LogDir, fmt.Sprintf("speech_%s%s", ts, ext))

	dst, err := os.Create(audioPath)
	if err != nil {
		jsonError(w, "save audio failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		jsonError(w, "save audio failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	dst.Close()

	// Transcribe
	text, err := s.Whisper.Transcribe(audioPath)
	if err != nil {
		log.Printf("[speech] transcribe error: %v", err)
		jsonError(w, "transcription failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[speech] Transcribed: %s", text)

	// Optionally send voice command to robot
	if text != "" {
		rb := s.Manager.GetCurrentRobot()
		if rb != nil && rb.Client != nil && rb.Client.IsConnected() {
			go rb.Client.SendVoiceCommand(text)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"text":   text,
		"status": "ok",
	})
}
