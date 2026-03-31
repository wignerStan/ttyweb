package config

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

var reloadMu sync.Mutex

// Reload re-reads the config file at path and atomically swaps the global config.
// After Reload, subsequent calls to Get() will return the updated config.
func Reload(path string) error {
	reloadMu.Lock()
	defer reloadMu.Unlock()

	cfg, err := Load(path)
	if err != nil {
		return err
	}
	globalConfig = cfg
	return nil
}

// Watch monitors path for changes and publishes new Config values on the returned
// channel. The caller must call the returned stop function when done to release
// the underlying file watcher.
func Watch(path string) (<-chan *Config, func(), error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving config path: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, fmt.Errorf("creating file watcher: %w", err)
	}

	if err := watcher.Add(absPath); err != nil {
		closeErr := watcher.Close()
		if closeErr != nil {
			return nil, nil, fmt.Errorf("adding watch path: %v (close error: %w)", err, closeErr)
		}
		return nil, nil, fmt.Errorf("adding watch path: %w", err)
	}

	ch := make(chan *Config, 1)
	done := make(chan struct{})

	go func() {
		defer close(ch)
		defer func() {
			// Watcher close errors are non-fatal and typically
			// indicate the watcher was already closed.
			_ = watcher.Close()
		}()
		runWatcherLoop(watcher, absPath, ch, done)
	}()

	stop := func() {
		close(done)
	}

	return ch, stop, nil
}

// runWatcherLoop reads from the fsnotify watcher and sends config updates.
// Extracted from Watch to reduce cognitive complexity.
func runWatcherLoop(watcher *fsnotify.Watcher, absPath string, ch chan<- *Config, done <-chan struct{}) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if err := Reload(absPath); err == nil {
					select {
					case ch <- Get():
					default:
					}
				}
			}
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		case <-done:
			return
		}
	}
}
