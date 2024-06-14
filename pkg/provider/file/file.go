package file

import (
	"http-benchmark/pkg/domain"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type ChangeFunc func() error

type ContentInfo struct {
	Content string
	Path    string
}
type FileProvider struct {
	opts      domain.FileProviderOptions
	watcher   *fsnotify.Watcher
	OnChanged ChangeFunc
}

func NewProvider(opts domain.FileProviderOptions) *FileProvider {
	return &FileProvider{
		opts: opts,
	}
}

func (p *FileProvider) Reset() {
	p.opts.Paths = p.opts.Paths[:0]
}

func (p *FileProvider) Add(path string) {
	p.opts.Paths = append(p.opts.Paths, path)
}

func (p *FileProvider) Open() ([]*ContentInfo, error) {
	p.opts.Paths = removeDuplicates(p.opts.Paths)

	var contents []*ContentInfo

	for _, path := range p.opts.Paths {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				content, err := os.ReadFile(filePath)
				if err != nil {
					return err
				}
				contents = append(contents, &ContentInfo{
					Content: string(content),
					Path:    filePath,
				})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return contents, nil
}

func (p *FileProvider) Watch() error {
	var err error

	if len(p.opts.Paths) == 0 {
		return nil
	}

	p.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func(watcher *fsnotify.Watcher) {
		defer watcher.Close()

		isUpdate := false
		refresh := 999 * time.Millisecond
		timer := time.NewTimer(refresh)
		defer timer.Stop()

		for {
			timer.Reset(refresh)

			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				switch {
				// 1. The Write, Create, and Rename events will be fired when saving the data, depending on the text editor you are using. For example, vi uses Rename
				// 2. When a large amount of data is being saved, the Write event will be triggered multiple times. Hence, we utilize a ticker and the 'isUpdate' parameter here.
				case event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write:
					isUpdate = true
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					// Some editors will remove the path from the watch list when the event is triggered, so we need to re-add it
					for _, path := range p.opts.Paths {
						err := p.addWatch(path)
						if err != nil {
							slog.Error("file watcher error", "error:", err)
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Error("file watcher error", "error:", err)
			case <-timer.C:
				if !isUpdate {
					continue
				}
				isUpdate = false

				if p.OnChanged != nil {
					err := p.OnChanged()
					if err != nil {
						slog.Error("Error in OnChanged:", "error:", err)
					}
				}
			}

		}
	}(p.watcher)

	for _, path := range p.opts.Paths {
		err := p.addWatch(path)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *FileProvider) addWatch(path string) error {
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		return p.watcher.Add(filePath)
	})
}

func removeDuplicates(strings []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, str := range strings {
		if _, found := seen[str]; !found {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}
