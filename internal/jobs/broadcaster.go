package jobs

import (
	"sync"
)

// SSEBroadcaster implements the Broadcaster interface
type SSEBroadcaster struct {
	subscribers map[string][]chan JobEvent
	mu          sync.RWMutex
	closed      bool
}

// NewSSEBroadcaster creates a new SSE broadcaster
func NewSSEBroadcaster() *SSEBroadcaster {
	return &SSEBroadcaster{
		subscribers: make(map[string][]chan JobEvent),
	}
}

// Subscribe creates a new subscription for job events
// Returns a channel to receive events and a cleanup function.
func (b *SSEBroadcaster) Subscribe(jobID string) (<-chan JobEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		// Return a closed channel if broadcaster is closed
		ch := make(chan JobEvent)
		close(ch)
		return ch, func() {}
	}

	// Create buffered channel to prevent blocking
	ch := make(chan JobEvent, 10)

	// Add to subscribers
	b.subscribers[jobID] = append(b.subscribers[jobID], ch)

	// Cleanup function
	cleanup := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		// Remove this channel from subscribers
		if subs, ok := b.subscribers[jobID]; ok {
			for i, sub := range subs {
				if sub == ch {
					// Remove from slice
					b.subscribers[jobID] = append(subs[:i], subs[i+1:]...)
					break
				}
			}

			// Remove job entry if no more subscribers
			if len(b.subscribers[jobID]) == 0 {
				delete(b.subscribers, jobID)
			}
		}

		// Close the channel
		close(ch)
	}

	return ch, cleanup
}

// Publish broadcasts an event to all subscribers of a job
func (b *SSEBroadcaster) Publish(event JobEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	// Send to all subscribers of this job
	if subs, ok := b.subscribers[event.JobID]; ok {
		for _, ch := range subs {
			// Non-blocking send
			select {
			case ch <- event:
			default:
				// Channel buffer is full, skip this subscriber
				// In production, you might want to log this
			}
		}
	}
}

// Close closes the broadcaster and all subscriber channels
func (b *SSEBroadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true

	// Close all subscriber channels
	for _, subs := range b.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}

	// Clear subscribers map
	b.subscribers = make(map[string][]chan JobEvent)
}

// SubscriberCount returns the number of active subscribers for a job
func (b *SSEBroadcaster) SubscriberCount(jobID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subs, ok := b.subscribers[jobID]; ok {
		return len(subs)
	}

	return 0
}

// TotalSubscribers returns the total number of subscribers across all jobs
func (b *SSEBroadcaster) TotalSubscribers() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	total := 0
	for _, subs := range b.subscribers {
		total += len(subs)
	}

	return total
}
