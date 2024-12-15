package file

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nite-coder/bifrost/pkg/provider"
)

type ContentInfo struct {
	Content string
	Path    string
}

type Options struct {
	Enabled    bool     `yaml:"enabled" json:"enabled"`
	Paths      []string `yaml:"paths" json:"paths"`
	Extensions []string `yaml:"extensions" json:"extensions"`
}

type FileProvider struct {
	options   Options
	watcher   *fsnotify.Watcher
	OnChanged provider.ChangeFunc
}

func NewProvider(opts Options) *FileProvider {
	if len(opts.Extensions) == 0 {
		opts.Extensions = []string{".yaml", ".yml", ".json"}
	}

	return &FileProvider{
		options: opts,
	}
}

func (p *FileProvider) Reset() {
	p.options.Paths = p.options.Paths[:0]
}

func (p *FileProvider) Add(path string) {
	p.options.Paths = append(p.options.Paths, path)
}

func (p *FileProvider) Open() ([]*ContentInfo, error) {
	p.options.Paths = removeDuplicates(p.options.Paths)

	var contents []*ContentInfo

	for _, path := range p.options.Paths {
		err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() {
				fileExtension := filepath.Ext(filePath)

				if len(fileExtension) == 0 {
					return nil
				}

				if !slices.Contains(p.options.Extensions, fileExtension) {
					return nil
				}

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

func (p *FileProvider) SetOnChanged(changeFunc provider.ChangeFunc) {
	p.OnChanged = changeFunc
}

func (p *FileProvider) Watch() error {
	var err error

	if len(p.options.Paths) == 0 {
		return nil
	}

	p.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func(watcher *fsnotify.Watcher) {
		defer watcher.Close()

		isUpdate := false
		refresh := 900 * time.Millisecond
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
					for _, path := range p.options.Paths {
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
						slog.Error("fail to change in file provider", "error:", err)
					}
				}
			}

		}
	}(p.watcher)

	for _, path := range p.options.Paths {
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
