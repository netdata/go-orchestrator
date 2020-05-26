package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/netdata/go-orchestrator/job/confgroup"

	"github.com/fsnotify/fsnotify"
)

type (
	Watcher struct {
		paths        []string
		reg          confgroup.Registry
		watcher      *fsnotify.Watcher
		cache        cache
		refreshEvery time.Duration
	}
	cache map[string]time.Time
)

func (c cache) lookup(path string) (time.Time, bool) { v, ok := c[path]; return v, ok }
func (c cache) remove(path string)                   { delete(c, path) }
func (c cache) put(path string, modTime time.Time)   { c[path] = modTime }

func NewWatcher(reg confgroup.Registry, paths []string) *Watcher {
	d := &Watcher{
		paths:        paths,
		reg:          reg,
		watcher:      nil,
		cache:        make(cache),
		refreshEvery: time.Minute,
	}
	return d
}

func (w Watcher) String() string {
	return "file watcher"
}

func (w *Watcher) Run(ctx context.Context, in chan<- []*confgroup.Group) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}

	w.watcher = watcher
	defer w.stop()
	w.refresh(ctx, in)

	tk := time.NewTicker(w.refreshEvery)
	defer tk.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			w.refresh(ctx, in)
		case event := <-w.watcher.Events:
			if event.Name == "" || isChmod(event) {
				break
			}
			if isRename(event) {
				// It is common to modify files using vim.
				// When writing to a file a backup is made. "backupcopy" option tells how it's done.
				// Default is "no": rename the file and write a new one.
				// This is cheap attempt to not send empty group for the old file.
				time.Sleep(time.Millisecond * 100)
			}
			w.refresh(ctx, in)
		case err := <-w.watcher.Errors:
			if err != nil {
			}
		}
	}
}

func (w *Watcher) listFiles() (files []string) {
	for _, pattern := range w.paths {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		files = append(files, matches...)
	}
	return files
}

func (w *Watcher) refresh(ctx context.Context, in chan<- []*confgroup.Group) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	var added, removed []*confgroup.Group
	seen := make(map[string]bool)

	for _, file := range w.listFiles() {
		fi, err := os.Lstat(file)
		if err != nil {
			continue
		}

		if !fi.Mode().IsRegular() {
			continue
		}

		seen[file] = true
		if v, ok := w.cache.lookup(file); ok && v.Equal(fi.ModTime()) {
			continue
		}
		w.cache.put(file, fi.ModTime())

		group, err := parseFile(w.reg, file)
		if err != nil {
			continue
		}
		added = append(added, group)
	}

	for name := range w.cache {
		if seen[name] {
			continue
		}
		w.cache.remove(name)
		removed = append(removed, &confgroup.Group{Source: name})
	}

	sendGroups(ctx, in, append(added, removed...))
	w.watchDirs()
}

func (w *Watcher) watchDirs() {
	for _, path := range w.paths {
		if idx := strings.LastIndex(path, "/"); idx > -1 {
			path = path[:idx]
		} else {
			path = "./"
		}
		if err := w.watcher.Add(path); err != nil {
		}
	}
}

func (w *Watcher) stop() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// closing the watcher deadlocks unless all events and errors are drained.
	go func() {
		for {
			select {
			case <-w.watcher.Errors:
			case <-w.watcher.Events:
			case <-ctx.Done():
				return
			}
		}
	}()

	if err := w.watcher.Close(); err != nil {
	}
}

func isChmod(event fsnotify.Event) bool {
	return event.Op^fsnotify.Chmod == 0
}

func isRename(event fsnotify.Event) bool {
	return event.Op&fsnotify.Rename == fsnotify.Rename
}

func sendGroups(ctx context.Context, in chan<- []*confgroup.Group, groups []*confgroup.Group) {
	if len(groups) == 0 {
		return
	}
	select {
	case <-ctx.Done():
	case in <- groups:
	}
}
