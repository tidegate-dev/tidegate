package certreload

import (
	"context"

	"github.com/fsnotify/fsnotify"

	"github.com/tidegate-dev/tidegate/internal/logging"
)

// dirWatcher watches directories with fsnotify. Kubernetes replaces a
// Secret's files by swapping a symlink next to them, so it watches the
// directories and not the files. While it's stopped, its channels are nil,
// and a select never picks them.
type dirWatcher struct {
	dirs       []string
	newWatcher func() (*fsnotify.Watcher, error)
	watcher    *fsnotify.Watcher
}

// start watches the directories, unless it already does. If that fails, it
// logs the error and stays stopped.
func (w *dirWatcher) start(ctx context.Context) {
	if w.watcher != nil {
		return
	}

	watcher, err := w.newWatcher()
	if err != nil {
		w.logFailure(ctx, err)

		return
	}

	for _, dir := range w.dirs {
		err := watcher.Add(dir)
		if err != nil {
			_ = watcher.Close()

			w.logFailure(ctx, err)

			return
		}
	}

	w.watcher = watcher
}

// stop logs err and stops watching until the next start.
func (w *dirWatcher) stop(ctx context.Context, err error) {
	w.logFailure(ctx, err)
	w.close()
}

func (w *dirWatcher) close() {
	if w.watcher != nil {
		_ = w.watcher.Close()
		w.watcher = nil
	}
}

func (w *dirWatcher) events() <-chan fsnotify.Event {
	if w.watcher == nil {
		return nil
	}

	return w.watcher.Events
}

func (w *dirWatcher) errors() <-chan error {
	if w.watcher == nil {
		return nil
	}

	return w.watcher.Errors
}

func (w *dirWatcher) logFailure(ctx context.Context, err error) {
	logging.FromContext(ctx).Error("watching the TLS certificate's files, checking them every minute instead", "dirs", w.dirs, "error", err)
}
