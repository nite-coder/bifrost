package pricing

import (
	"embed"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"github.com/nite-coder/bifrost/pkg/config"
)

//go:embed prices.json
var embeddedFiles embed.FS

var (
	registry map[string]*config.AIPricingOptions
	mu       sync.RWMutex
)

// Init initializes the pricing registry. It loads embedded defaults first,
// and then merges/overrides them with prices from customPath if provided.
func Init(customPath string) error {
	mu.Lock()
	defer mu.Unlock()

	// Load embedded defaults
	data, err := embeddedFiles.ReadFile("prices.json")
	if err != nil {
		return fmt.Errorf("failed to read embedded prices.json: %w", err)
	}

	var embeddedRegistry map[string]*config.AIPricingOptions
	if err := json.Unmarshal(data, &embeddedRegistry); err != nil {
		return fmt.Errorf("failed to unmarshal embedded prices: %w", err)
	}

	registry = embeddedRegistry

	// Load custom overrides if provided
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			customData, err := os.ReadFile(filepath.Clean(customPath))
			if err != nil {
				return fmt.Errorf("failed to read custom pricing file: %w", err)
			}

			var customRegistry map[string]*config.AIPricingOptions
			if err := json.Unmarshal(customData, &customRegistry); err != nil {
				return fmt.Errorf("failed to unmarshal custom prices: %w", err)
			}

			// Merge custom prices into registry
			maps.Copy(registry, customRegistry)
		}
	}

	return nil
}

// Resolve returns the pricing options for a given handler and model.
// Priority:
// 1. override (if not nil)
// 2. handler/model
// 3. model
// Returns nil if not found.
func Resolve(handler, model string, override *config.AIPricingOptions) *config.AIPricingOptions {
	if override != nil {
		return override
	}

	mu.RLock()
	defer mu.RUnlock()

	if registry == nil {
		return nil
	}

	// Try handler/model
	key := handler + "/" + model
	if p, ok := registry[key]; ok {
		return p
	}

	// Fallback to model
	if p, ok := registry[model]; ok {
		return p
	}

	return nil
}
