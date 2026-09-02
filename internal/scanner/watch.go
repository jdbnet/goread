package scanner

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (s *Scanner) Watch(stop <-chan struct{}) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("fsnotify disabled: %v", err)
		return
	}
	defer w.Close()

	_ = filepath.WalkDir(s.Library, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") && path != s.Library {
			return filepath.SkipDir
		}
		if err := w.Add(path); err != nil {
			log.Printf("fsnotify watch %s: %v", path, err)
		}
		return nil
	})

	debounce := time.NewTimer(time.Hour)
	if !debounce.Stop() {
		select {
		case <-debounce.C:
		default:
		}
	}
	pending := false
	kick := func() {
		pending = true
		if !debounce.Stop() {
			select {
			case <-debounce.C:
			default:
			}
		}
		debounce.Reset(2 * time.Second)
	}

	for {
		select {
		case <-stop:
			return
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			if ev.Has(fsnotify.Create) {
				fi, err := os.Stat(ev.Name)
				if err == nil && fi.IsDir() {
					_ = w.Add(ev.Name)
				}
			}
			kick()
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			log.Printf("fsnotify: %v", err)
		case <-debounce.C:
			if !pending {
				continue
			}
			pending = false
			if _, err := s.Scan(); err != nil && err.Error() != "scan already running" {
				log.Printf("fsnotify scan: %v", err)
			}
		}
	}
}
