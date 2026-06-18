package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// StreamHandler serves HLS files (m3u8 playlists and .ts segments)
type StreamHandler struct {
	streamsDir string
}

// NewStreamHandler creates a new StreamHandler
func NewStreamHandler(streamsDir string) *StreamHandler {
	return &StreamHandler{
		streamsDir: streamsDir,
	}
}

// ServeHTTP handles GET /api/stream/{id}/*
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Get the rest of the path after /api/stream/{id}/
	filePath := chi.URLParam(r, "*")

	if id == "" || filePath == "" {
		http.NotFound(w, r)
		return
	}

	// Prevent path traversal
	cleanPath := filepath.Clean(filePath)
	if strings.Contains(cleanPath, "..") {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join(h.streamsDir, id, cleanPath)

	// Set appropriate Content-Type
	switch {
	case strings.HasSuffix(fullPath, ".m3u8"):
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	case strings.HasSuffix(fullPath, ".ts"):
		w.Header().Set("Content-Type", "video/mp2t")
	}

	// CORS headers for cross-origin playback
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Cache headers — segments are immutable
	if strings.HasSuffix(fullPath, ".ts") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	http.ServeFile(w, r, fullPath)
}
