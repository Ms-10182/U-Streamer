package worker

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hls-streaming-server/internal/ffmpeg"
	"github.com/hls-streaming-server/internal/queue"
)

// Worker processes video transcoding jobs from a queue
type Worker struct {
	queue      *queue.Queue
	uploadsDir string
	streamsDir string
}

// New creates a new Worker
func New(q *queue.Queue, uploadsDir, streamsDir string) *Worker {
	return &Worker{
		queue:      q,
		uploadsDir: uploadsDir,
		streamsDir: streamsDir,
	}
}

// Start begins consuming jobs from the queue. Blocks until the channel is closed.
// Should be run in a goroutine.
func (w *Worker) Start() {
	log.Println("[WORKER] Started, waiting for jobs...")

	for job := range w.queue.Jobs() {
		w.processJob(job)
	}

	log.Println("[WORKER] Queue closed, shutting down")
}

func (w *Worker) processJob(job queue.Job) {
	log.Printf("[WORKER] Processing job: %s (file: %s)", job.ID, job.FilePath)

	// 1. Verify the uploaded file exists
	if _, err := os.Stat(job.FilePath); os.IsNotExist(err) {
		log.Printf("[WORKER] ERROR: File not found: %s", job.FilePath)
		return
	}

	// 2. Probe the video to get resolution
	info, err := ffmpeg.Probe(job.FilePath)
	if err != nil {
		log.Printf("[WORKER] ERROR: Failed to probe %s: %v", job.FilePath, err)
		return
	}

	log.Printf("[WORKER] Video resolution: %dx%d", info.Width, info.Height)

	// 3. Select appropriate renditions (never upscale)
	renditions := ffmpeg.SelectRenditions(info.Width, info.Height)
	renditionNames := make([]string, len(renditions))
	for i, r := range renditions {
		renditionNames[i] = r.Name
	}
	log.Printf("[WORKER] Selected renditions: %s", strings.Join(renditionNames, ", "))

	// 4. Create output directory
	outputDir := filepath.Join(w.streamsDir, job.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("[WORKER] ERROR: Failed to create output dir: %v", err)
		return
	}

	// 5. Write a processing status marker
	statusFile := filepath.Join(outputDir, ".processing")
	os.WriteFile(statusFile, []byte("processing"), 0644)

	// 6. Transcode
	if err := ffmpeg.Transcode(job.FilePath, outputDir, renditions); err != nil {
		log.Printf("[WORKER] ERROR: Transcoding failed for %s: %v", job.ID, err)
		return
	}

	// 7. Remove processing marker
	os.Remove(statusFile)

	// 8. Write metadata
	metaContent := fmt.Sprintf("source_width=%d\nsource_height=%d\nrenditions=%s\n",
		info.Width, info.Height, strings.Join(renditionNames, ","))
	metaFile := filepath.Join(outputDir, "metadata.txt")
	os.WriteFile(metaFile, []byte(metaContent), 0644)

	log.Printf("[WORKER] ✅ Job complete: %s — %d renditions created", job.ID, len(renditions))
}
