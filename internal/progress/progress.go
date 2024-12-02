package progress

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Tracker handles progress tracking for file operations
type Tracker struct {
	total     int64
	current   int64
	startTime time.Time
}

// NewTracker creates a new progress tracker
func NewTracker(total int64) *Tracker {
	return &Tracker{
		total:     total,
		current:   0,
		startTime: time.Now(),
	}
}

// Update updates the current progress
func (t *Tracker) Update(n int64) {
	atomic.AddInt64(&t.current, n)
}

// GetProgress returns the current progress percentage
func (t *Tracker) GetProgress() float64 {
	current := atomic.LoadInt64(&t.current)
	return float64(current) / float64(t.total) * 100
}

// GetSpeed returns the current speed in bytes per second
func (t *Tracker) GetSpeed() float64 {
	current := atomic.LoadInt64(&t.current)
	duration := time.Since(t.startTime).Seconds()
	if duration > 0 {
		return float64(current) / duration
	}
	return 0
}

// GetETA returns the estimated time remaining
func (t *Tracker) GetETA() time.Duration {
	current := atomic.LoadInt64(&t.current)
	if current == 0 {
		return time.Duration(0)
	}

	speed := t.GetSpeed()
	if speed == 0 {
		return time.Duration(0)
	}

	remaining := t.total - current
	seconds := float64(remaining) / speed
	return time.Duration(seconds * float64(time.Second))
}

// String returns a formatted progress string
func (t *Tracker) String() string {
	progress := t.GetProgress()
	speed := t.GetSpeed()
	eta := t.GetETA()

	return fmt.Sprintf("%.1f%% (%.2f MB/s) ETA: %v",
		progress,
		speed/1024/1024,
		eta.Round(time.Second),
	)
}
