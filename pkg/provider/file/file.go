package file

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/provider"
)

// ContentInfo holds the content of a file and its path.
type ContentInfo struct {
	Content string
	Path    string
}

// Options defines the configuration for the file provider.
type Options struct {
	Paths      []string `json:"paths"      yaml:"paths"`
	Extensions []string `json:"extensions" yaml:"extensions"`
	Watch      bool     `json:"watch"      yaml:"watch"`
	Enabled    bool     `json:"enabled"    yaml:"enabled"`
}

// Provider implements a configuration provider that reads from the local filesystem.
type Provider struct {
	watcher   *fsnotify.Watcher
	OnChanged provider.ChangeFunc
	options   Options
}

// NewProvider creates a new FileProvider instance.
func NewProvider(opts Options) *Provider {
	if len(opts.Extensions) == 0 {
		opts.Extensions = []string{".yaml", ".yml"}
	}
	return &Provider{
		options: opts,
	}
}

// Reset clears the paths in the file provider.
func (p *Provider) Reset() {
	p.options.Paths = p.options.Paths[:0]
}

// Add adds a path to the file provider.
func (p *Provider) Add(path string) {
	p.options.Paths = append(p.options.Paths, path)
}

// Open reads all files from the configured paths and returns their content.
func (p *Provider) Open() ([]*ContentInfo, error) {
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
						/* #nosec G304 */
						content, err := os.ReadFile(filepath.Clean(filePath))
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
			/* #nosec G304 */
			content, err := os.ReadFile(filepath.Clean(path))
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

// SetOnChanged sets the callback function to be called when a file changes.
func (p *Provider) SetOnChanged(changeFunc provider.ChangeFunc) {
	p.OnChanged = changeFunc
}

// Watch starts watching the configured paths for changes.
func (p *Provider) Watch() error {
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
		go safety.Go(context.Background(), func() {
			defer watcher.Close()
			isUpdate := false
			const defaultRefreshInterval = 900 * time.Millisecond
			timer := time.NewTimer(defaultRefreshInterval)
			defer timer.Stop()
			for {
				timer.Reset(defaultRefreshInterval)
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
					default:
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

func (p *Provider) addWatch(path string) error {
	return filepath.Walk(path, func(filePath string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return p.watcher.Add(filePath)
	})
}
