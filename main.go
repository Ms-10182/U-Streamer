package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hls-streaming-server/internal/handler"
	"github.com/hls-streaming-server/internal/queue"
	"github.com/hls-streaming-server/internal/worker"
)

const (
	uploadsDir = "./uploads"
	streamsDir = "./streams"
	port       = ":8080"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("🎬 HLS Streaming Server starting up...")

	// Create required directories
	for _, dir := range []string{uploadsDir, streamsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Initialize job queue
	jobQueue := queue.New(100)

	// Start background worker
	w := worker.New(jobQueue, uploadsDir, streamsDir)
	go w.Start()
	log.Println("✅ Background worker started")

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	// API routes
	r.Post("/api/upload", handler.NewUploadHandler(uploadsDir, jobQueue).ServeHTTP)
	r.Get("/api/videos", handler.NewVideosHandler(streamsDir).ServeHTTP)
	r.Get("/api/stream/{id}/*", handler.NewStreamHandler(streamsDir).ServeHTTP)

	// Frontend routes
	r.Get("/", serveFile("web/index.html"))
	r.Get("/player", serveFile("web/player.html"))

	// Static files
	r.Handle("/web/*", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

	log.Printf("🚀 Server listening on http://localhost%s", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func serveFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
