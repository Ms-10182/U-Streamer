package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/hls-streaming-server/internal/queue"
)

// UploadHandler handles video file uploads
type UploadHandler struct {
	uploadsDir string
	queue      *queue.Queue
}

// NewUploadHandler creates a new UploadHandler
func NewUploadHandler(uploadsDir string, q *queue.Queue) *UploadHandler {
	return &UploadHandler{
		uploadsDir: uploadsDir,
		queue:      q,
	}
}

// uploadResponse is the JSON response for a successful upload
type uploadResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// errorResponse is the JSON response for errors
type errorResponse struct {
	Error string `json:"error"`
}

// ServeHTTP handles POST /api/upload
func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	// Limit upload size to 2GB
	r.Body = http.MaxBytesReader(w, r.Body, 2<<30)

	// Parse multipart form
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100MB in memory, rest on disk
		log.Printf("[UPLOAD] Failed to parse form: %v", err)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "failed to parse upload. max size is 2GB"})
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("video")
	if err != nil {
		log.Printf("[UPLOAD] No file in request: %v", err)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "no video file provided. use form field 'video'"})
		return
	}
	defer file.Close()

	// Generate UUID
	id := uuid.New().String()

	// Get file extension
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".mp4" // default
	}

	// Save to disk
	destPath := filepath.Join(h.uploadsDir, id+ext)
	destFile, err := os.Create(destPath)
	if err != nil {
		log.Printf("[UPLOAD] Failed to create file: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to save file"})
		return
	}
	defer destFile.Close()

	written, err := io.Copy(destFile, file)
	if err != nil {
		log.Printf("[UPLOAD] Failed to write file: %v", err)
		os.Remove(destPath) // cleanup
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to save file"})
		return
	}

	log.Printf("[UPLOAD] Saved %s (%s, %d bytes) as %s", header.Filename, formatBytes(written), written, id+ext)

	// Enqueue for processing
	h.queue.Enqueue(queue.Job{
		ID:       id,
		FilePath: destPath,
	})

	// Create stream directory with processing marker
	streamDir := filepath.Join(filepath.Dir(h.uploadsDir), "streams", id)
	os.MkdirAll(streamDir, 0755)
	os.WriteFile(filepath.Join(streamDir, ".processing"), []byte("queued"), 0644)

	writeJSON(w, http.StatusOK, uploadResponse{
		ID:     id,
		Status: "queued",
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
