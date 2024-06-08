package file

import (
	"fmt"
	"http-benchmark/pkg/domain"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type FileProvider struct {
}

func NewFileProvider() *FileProvider {
	return &FileProvider{}
}

func (p *FileProvider) Open(path string) (domain.Options, error) {
	opts := domain.Options{}

	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return opts, fmt.Errorf("file: read file error: %w", err)
	}

	err = yaml.Unmarshal(content, &opts)
	if err != nil {
		return opts, fmt.Errorf("file: yaml unmarshal failed. err: %w", err)
	}



	
	return opts, nil
}

func fileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}
