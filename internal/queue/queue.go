package queue

import (
	"log"
	"sync"
)

// Job represents a video processing job
type Job struct {
	ID       string // UUID of the video
	FilePath string // Full path to the uploaded file
}

// Queue is a thread-safe job queue backed by a Go channel
type Queue struct {
	jobs chan Job
	mu   sync.Mutex
}

// New creates a new Queue with the given buffer size
func New(bufferSize int) *Queue {
	return &Queue{
		jobs: make(chan Job, bufferSize),
	}
}

// Enqueue adds a job to the queue. Non-blocking if buffer has space.
func (q *Queue) Enqueue(job Job) {
	q.mu.Lock()
	defer q.mu.Unlock()

	select {
	case q.jobs <- job:
		log.Printf("[QUEUE] Enqueued job: %s", job.ID)
	default:
		log.Printf("[QUEUE] WARNING: Queue is full, dropping job: %s", job.ID)
	}
}

// Jobs returns the channel for consuming jobs
func (q *Queue) Jobs() <-chan Job {
	return q.jobs
}

// Len returns the current number of jobs in the queue
func (q *Queue) Len() int {
	return len(q.jobs)
}
