package watcher

import (
	"fmt"
	"path/filepath"
	"time"

	"os"

	"github.com/fsnotify/fsnotify"
)

type FileEvent struct {
	Path      string
	Operation string
	Time      time.Time
}

type Watcher struct {
	watcher    *fsnotify.Watcher
	events     chan FileEvent
	errors     chan error
	done       chan struct{}
	debounceMs int
}

func NewWatcher(debounceMs int) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &Watcher{
		watcher:    fsWatcher,
		events:     make(chan FileEvent),
		errors:     make(chan error),
		done:       make(chan struct{}),
		debounceMs: debounceMs,
	}, nil
}

func (w *Watcher) Watch(path string, recursive bool) error {
	if recursive {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return w.watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}
	} else {
		if err := w.watcher.Add(path); err != nil {
			return fmt.Errorf("failed to watch path: %w", err)
		}
	}

	go w.processEvents()
	return nil
}

func (w *Watcher) processEvents() {
	eventMap := make(map[string]FileEvent)
	timer := time.NewTimer(time.Duration(w.debounceMs) * time.Millisecond)
	timer.Stop()

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			eventMap[event.Name] = FileEvent{
				Path:      event.Name,
				Operation: event.Op.String(),
				Time:      time.Now(),
			}
			timer.Reset(time.Duration(w.debounceMs) * time.Millisecond)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.errors <- err

		case <-timer.C:
			for _, event := range eventMap {
				w.events <- event
			}
			eventMap = make(map[string]FileEvent)

		case <-w.done:
			return
		}
	}
}

func (w *Watcher) Events() <-chan FileEvent {
	return w.events
}

func (w *Watcher) Errors() <-chan error {
	return w.errors
}

func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}
