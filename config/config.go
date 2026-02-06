package config

import (
	"os"
	"path/filepath"
)

// Config holds application configuration.
type Config struct {
	ListenAddr        string
	RosbridgePort     int
	WhisperBinPath    string
	WhisperModelPath  string
	SpeechLogDir      string
	DefaultLinearMax  float64
	DefaultAngularMax float64
}

// Load returns configuration from environment or defaults.
func Load() *Config {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/home/data"
	}

	whisperBin := envOr("WHISPER_BIN", filepath.Join(home, "data/app/whisper.cpp/build/bin/whisper-cli"))
	whisperModel := envOr("WHISPER_MODEL", filepath.Join(home, "data/app/whisper.cpp/models/ggml-base.en.bin"))
	speechDir := envOr("SPEECH_LOG_DIR", filepath.Join(home, "data/log/wav"))

	return &Config{
		ListenAddr:        envOr("LISTEN_ADDR", ":8080"),
		RosbridgePort:     9090,
		WhisperBinPath:    whisperBin,
		WhisperModelPath:  whisperModel,
		SpeechLogDir:      speechDir,
		DefaultLinearMax:  1.0,
		DefaultAngularMax: 1.0,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
