package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/netdata/go-orchestrator/job/confgroup"

	"github.com/fsnotify/fsnotify"
)

type (
	Watcher struct {
		paths   []string
		reg     confgroup.Registry
		watcher *fsnotify.Watcher
		cache   cache
	}
	cache map[string]struct{}
)

func (c cache) has(path string) bool { _, ok := c[path]; return ok }
func (c cache) remove(path string)   { delete(c, path) }
func (c cache) put(path string)      { c[path] = struct{}{} }

func NewWatcher(reg confgroup.Registry, paths []string) *Watcher {
	d := &Watcher{
		paths:   paths,
		reg:     reg,
		watcher: nil,
		cache:   make(cache),
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
	w.refreshFiles(ctx, in)

	tk := time.NewTicker(time.Second * 5)
	defer tk.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			w.refreshFiles(ctx, in)
		case event := <-w.watcher.Events:
			w.processEvent(ctx, event, in)
		case err := <-w.watcher.Errors:
			if err != nil {
			}
		}
	}
}

func (w *Watcher) refreshFiles(ctx context.Context, in chan<- []*confgroup.Group) {
	var groups []*confgroup.Group
	for _, pattern := range w.paths {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, path := range matches {
			if w.cache.has(path) {
				continue
			}

			if ok, err := isFile(path); !ok || err != nil {
				continue
			}

			group, err := readFile(w.reg, path)
			if err != nil {
				continue
			}
			// TODO: this order is probably wrong
			w.cache.put(path)
			groups = append(groups, group)

			if err := w.watcher.Add(path); err != nil {
				fmt.Println(err)
				continue
			}

		}
	}
	sendGroups(ctx, in, groups)
}

func (w *Watcher) processEvent(ctx context.Context, event fsnotify.Event, in chan<- []*confgroup.Group) {
	if event.Name == "" || isChmod(event) {
		return
	}

	if isRenameOrRemove(event) {
		// TODO handle rename, should follow the file, not send empty group
		w.cache.remove(event.Name)
		sendGroup(ctx, in, &confgroup.Group{Source: event.Name})
		return
	}

	group, err := readFile(w.reg, event.Name)
	if err != nil {
		return
	}

	sendGroup(ctx, in, group)
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

func isRenameOrRemove(event fsnotify.Event) bool {
	return event.Op&fsnotify.Rename == fsnotify.Rename || event.Op&fsnotify.Remove == fsnotify.Remove
}

func isChmod(event fsnotify.Event) bool {
	return event.Op^fsnotify.Chmod == 0
}

func isFile(path string) (bool, error) {
	fi, err := os.Stat(path)
	return err == nil && fi != nil && fi.Mode().IsRegular(), err
}

func sendGroup(ctx context.Context, in chan<- []*confgroup.Group, group *confgroup.Group) {
	select {
	case <-ctx.Done():
	case in <- []*confgroup.Group{group}:
	}
}

func sendGroups(ctx context.Context, in chan<- []*confgroup.Group, groups []*confgroup.Group) {
	select {
	case <-ctx.Done():
	case in <- groups:
	}
}
