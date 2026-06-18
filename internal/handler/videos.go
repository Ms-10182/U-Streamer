package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// VideosHandler handles listing all videos
type VideosHandler struct {
	streamsDir string
}

// NewVideosHandler creates a new VideosHandler
func NewVideosHandler(streamsDir string) *VideosHandler {
	return &VideosHandler{
		streamsDir: streamsDir,
	}
}

// VideoItem represents a video in the listing
type VideoItem struct {
	ID         string   `json:"id"`
	Status     string   `json:"status"` // "processing", "ready"
	Renditions []string `json:"renditions,omitempty"`
}

// ServeHTTP handles GET /api/videos
func (h *VideosHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(h.streamsDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []VideoItem{})
			return
		}
		log.Printf("[VIDEOS] Failed to read streams dir: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list videos"})
		return
	}

	var videos []VideoItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		id := entry.Name()
		videoDir := filepath.Join(h.streamsDir, id)

		item := VideoItem{ID: id}

		// Check if processing marker exists
		if _, err := os.Stat(filepath.Join(videoDir, ".processing")); err == nil {
			item.Status = "processing"
		} else if _, err := os.Stat(filepath.Join(videoDir, "master.m3u8")); err == nil {
			item.Status = "ready"
			item.Renditions = detectRenditions(videoDir)
		} else {
			item.Status = "processing"
		}

		videos = append(videos, item)
	}

	if videos == nil {
		videos = []VideoItem{} // ensure non-null JSON
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(videos)
}

// detectRenditions scans the stream directory for rendition folders
func detectRenditions(videoDir string) []string {
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		return nil
	}

	var renditions []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Check if it looks like a rendition directory (has index.m3u8)
		if _, err := os.Stat(filepath.Join(videoDir, name, "index.m3u8")); err == nil {
			renditions = append(renditions, name)
		}
	}
	return renditions
}

// ReadMetadata reads the metadata.txt file for a video
func ReadMetadata(videoDir string) map[string]string {
	data, err := os.ReadFile(filepath.Join(videoDir, "metadata.txt"))
	if err != nil {
		return nil
	}

	meta := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			meta[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return meta
}
