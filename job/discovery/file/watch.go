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
func (c cache) put(fi os.FileInfo)                   { c[fi.Name()] = fi.ModTime() }

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
	var added, removed []*confgroup.Group
	seen := make(map[string]bool)

	for _, file := range w.listFiles() {
		fi, err := os.Stat(file)
		if err != nil {
			continue
		}

		if !fi.Mode().IsRegular() {
			continue
		}

		seen[fi.Name()] = true
		if v, ok := w.cache.lookup(fi.Name()); ok && v.Equal(fi.ModTime()) {
			continue
		}
		w.cache.put(fi)

		group, err := parseFile(w.reg, fi.Name())
		if err != nil {
			continue
		}
		added = append(added, group)
	}
	sendGroups(ctx, in, added)

	for name := range w.cache {
		if seen[name] {
			continue
		}
		w.cache.remove(name)
		removed = append(removed, &confgroup.Group{Source: name})
	}
	sendGroups(ctx, in, removed)

	w.watchDirs()
}

func (w *Watcher) watchDirs() {
	for _, p := range w.paths {
		if idx := strings.LastIndex(p, "/"); idx != -1 {
			p = p[:idx]
		} else {
			p = "./"
		}
		if err := w.watcher.Add(p); err != nil {
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

func sendGroups(ctx context.Context, in chan<- []*confgroup.Group, groups []*confgroup.Group) {
	if len(groups) == 0 {
		return
	}
	select {
	case <-ctx.Done():
	case in <- groups:
	}
}
