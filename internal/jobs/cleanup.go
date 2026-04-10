package jobs

import (
	"context"
	"log"
	"time"
)

// CleanupWorker runs periodic cleanup of expired jobs
type CleanupWorker struct {
	manager  *Manager
	interval time.Duration
	done     chan struct{}
}

// NewCleanupWorker creates a new cleanup worker
func NewCleanupWorker(manager *Manager, intervalMinutes int) *CleanupWorker {
	return &CleanupWorker{
		manager:  manager,
		interval: time.Duration(intervalMinutes) * time.Minute,
		done:     make(chan struct{}),
	}
}

// Start begins the cleanup worker goroutine
func (w *CleanupWorker) Start(ctx context.Context) {
	log.Printf("Starting job cleanup worker (interval: %v)", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run cleanup immediately on start
	w.cleanup(ctx)

	for {
		select {
		case <-ticker.C:
			w.cleanup(ctx)
		case <-ctx.Done():
			log.Println("Stopping job cleanup worker")
			close(w.done)
			return
		case <-w.done:
			return
		}
	}
}

// Stop stops the cleanup worker
func (w *CleanupWorker) Stop() {
	close(w.done)
}

// cleanup performs the actual cleanup operation
func (w *CleanupWorker) cleanup(ctx context.Context) {
	count, err := w.manager.DeleteExpiredJobs(ctx)
	if err != nil {
		log.Printf("Error during job cleanup: %v", err)
		return
	}

	if count > 0 {
		log.Printf("✓ Cleaned up %d expired jobs", count)
	}
}
