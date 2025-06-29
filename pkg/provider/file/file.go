package file

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/provider"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type ContentInfo struct {
	Content string
	Path    string
}
type Options struct {
	Paths      []string `yaml:"paths" json:"paths"`
	Extensions []string `yaml:"extensions" json:"extensions"`
	Watch      bool
	Enabled    bool `yaml:"enabled" json:"enabled"`
}
type FileProvider struct {
	watcher   *fsnotify.Watcher
	OnChanged provider.ChangeFunc
	options   Options
}

func NewProvider(opts Options) *FileProvider {
	if len(opts.Extensions) == 0 {
		opts.Extensions = []string{".yaml", ".yml"}
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
	p.options.Paths = slices.Compact(p.options.Paths)
	var contents []*ContentInfo
	for _, path := range p.options.Paths {
		// Check if the path is a file or a directory
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			// If it's a directory, read files in the directory (non-recursive)
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				if !entry.IsDir() { // Skip subdirectories
					filePath := filepath.Join(path, entry.Name())
					fileExtension := filepath.Ext(filePath)
					// Check if the file extension is in the allowed list
					if len(fileExtension) > 0 && slices.Contains(p.options.Extensions, fileExtension) {
						content, err := os.ReadFile(filePath)
						if err != nil {
							return nil, err
						}
						contents = append(contents, &ContentInfo{
							Content: string(content),
							Path:    filePath,
						})
					}
				}
			}
		} else {
			// If it's a file, read it directly
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			contents = append(contents, &ContentInfo{
				Content: string(content),
				Path:    path,
			})
		}
	}
	return contents, nil
}
func (p *FileProvider) SetOnChanged(changeFunc provider.ChangeFunc) {
	p.OnChanged = changeFunc
}
func (p *FileProvider) Watch() error {
	if !p.options.Watch {
		return nil
	}
	var err error
	if len(p.options.Paths) == 0 {
		return nil
	}
	p.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	go func(watcher *fsnotify.Watcher) {
		safety.Go(context.Background(), func() {
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
						isUpdate = true
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
							slog.Error("failed to change in file provider", "error:", err)
						}
					}
				}
			}
		})
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
