package file

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestDir(t *testing.T) (string, func()) {
	dir, err := os.MkdirTemp("", "fileprovider-test")
	require.NoError(t, err)

	files := []struct {
		path    string
		content string
	}{
		{"dir1/file1.yaml", "yaml content"},
		{"dir1/file2.json", "json content"},
		{"dir1/subdir/file3.txt", "txt content"},
		{"dir2/config.yml", "yml content"},
		{"single.json", "single content"},
	}

	for _, f := range files {
		path := filepath.Join(dir, f.path)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(f.content), 0644))
	}

	return dir, func() { os.RemoveAll(dir) }
}

func TestFileProviderOpen(t *testing.T) {
	testDir, cleanup := createTestDir(t)
	defer cleanup()

	tests := []struct {
		name        string
		paths       []string
		extensions  []string
		wantCount   int
		wantError   bool
		wantContent []string
	}{
		{
			name:        "Single valid file",
			paths:       []string{filepath.Join(testDir, "single.json")},
			extensions:  []string{".json"},
			wantCount:   1,
			wantContent: []string{"single content"},
		},
		{
			name:        "Directory with multiple extensions",
			paths:       []string{filepath.Join(testDir, "dir1")},
			extensions:  []string{".yaml", ".json"},
			wantCount:   2,
			wantContent: []string{"yaml content", "json content"},
		},
		{
			name:      "Non-existing path",
			paths:     []string{"/non/existing/path"},
			wantError: true,
		},
		{
			name:        "Filter by extension",
			paths:       []string{filepath.Join(testDir, "dir1/subdir/file3.txt")},
			extensions:  []string{".txt"},
			wantCount:   1,
			wantContent: []string{"txt content"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProvider(Options{
				Paths:      tt.paths,
				Extensions: tt.extensions,
			})

			contents, err := p.Open()

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, len(contents))

			var gotContents []string
			for _, c := range contents {
				gotContents = append(gotContents, c.Content)
			}
			for _, want := range tt.wantContent {
				assert.Contains(t, gotContents, want)
			}
		})
	}
}

func TestFileProvider_AddReset(t *testing.T) {
	p := NewProvider(Options{})
	assert.Empty(t, p.options.Paths)

	p.Add("/path1")
	p.Add("/path2")
	assert.Equal(t, []string{"/path1", "/path2"}, p.options.Paths)

	p.Reset()
	assert.Empty(t, p.options.Paths)
}

func TestFileProvider_OpenErrors(t *testing.T) {
	t.Run("Invalid directory", func(t *testing.T) {
		p := NewProvider(Options{Paths: []string{"/invalid/path"}})
		_, err := p.Open()
		assert.Error(t, err)
	})

	t.Run("Unreadable file", func(t *testing.T) {
		testDir, cleanup := createTestDir(t)
		defer cleanup()

		badFile := filepath.Join(testDir, "badfile.yaml")
		require.NoError(t, os.WriteFile(badFile, nil, 0000))

		p := NewProvider(Options{Paths: []string{badFile}})
		_, err := p.Open()
		assert.Error(t, err)
	})
}

func TestFileProvider_Watch(t *testing.T) {
	testDir, cleanup := createTestDir(t)
	defer cleanup()

	// create test file
	targetFile := filepath.Join(testDir, "watch_test.yaml")
	require.NoError(t, os.WriteFile(targetFile, []byte("initial content"), 0644))

	p := NewProvider(Options{
		Paths:      []string{targetFile},
		Extensions: []string{".yaml"},
	})

	var eventCounter atomic.Int32
	p.SetOnChanged(func() error {
		eventCounter.Add(1)
		return nil
	})

	// enable watch
	require.NoError(t, p.Watch())
	defer p.watcher.Close()

	// first modify
	require.NoError(t, os.WriteFile(targetFile, []byte("modified content 1"), 0644))
	time.Sleep(1 * time.Second)
	assert.Equal(t, int32(1), eventCounter.Load())

	// second modify
	require.NoError(t, os.WriteFile(targetFile, []byte("modified content 2"), 0644))
	time.Sleep(1 * time.Second)
	assert.Equal(t, int32(2), eventCounter.Load())

	// delete and create
	require.NoError(t, os.Remove(targetFile))
	time.Sleep(100 * time.Millisecond)
	require.NoError(t, os.WriteFile(targetFile, []byte("new content"), 0644))
	time.Sleep(1 * time.Second)
	assert.Equal(t, int32(3), eventCounter.Load())
}
