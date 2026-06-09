package pricing

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/nite-coder/bifrost/pkg/config"
)

func TestResolve(t *testing.T) {
	// Initialize with empty custom path to test embedded defaults
	err := Init("")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Test resolving embedded gpt-4o price
	p := Resolve("openai-chat", "gpt-4o", nil)
	if p == nil {
		t.Fatal("failed to resolve embedded gpt-4o price, got nil")
	}
	if p.InputPerMtok != 2.50 {
		t.Errorf("expected InputPerMtok 2.50, got %f", p.InputPerMtok)
	}

	// Test resolve with override (highest priority)
	override := &config.AIPricingOptions{InputPerMtok: 1.0}
	p2 := Resolve("openai-chat", "gpt-4o", override)
	if p2 == nil {
		t.Fatal("failed to resolve with override, got nil")
	}
	if p2.InputPerMtok != 1.0 {
		t.Errorf("expected InputPerMtok 1.0, got %f", p2.InputPerMtok)
	}

	// Test fallback (resolving with just model name)
	// We'll need to make sure deepseek-chat is in prices.json
	p3 := Resolve("some-other-handler", "deepseek-chat", nil)
	if p3 == nil {
		t.Fatal("failed to resolve deepseek-chat with fallback, got nil")
	}
	if p3.InputPerMtok != 0.14 {
		t.Errorf("expected InputPerMtok 0.14 for deepseek-chat fallback, got %f", p3.InputPerMtok)
	}

	// Test non-existent model
	p4 := Resolve("unknown", "unknown", nil)
	if p4 != nil {
		t.Errorf("expected nil for unknown model, got %+v", p4)
	}
}

func TestCustomPath(t *testing.T) {
	tmpDir := t.TempDir()

	customPricesPath := filepath.Join(tmpDir, "custom_prices.json")
	customContent := `{
		"openai-chat/gpt-4o": {
			"input_per_mtok": 5.0,
			"output_per_mtok": 10.0
		},
		"new-model": {
			"input_per_mtok": 0.5
		}
	}`
	if err := os.WriteFile(customPricesPath, []byte(customContent), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Init(customPricesPath); err != nil {
		t.Fatalf("Init with custom path failed: %v", err)
	}

	// Check override from custom file
	p := Resolve("openai-chat", "gpt-4o", nil)
	if p == nil || p.InputPerMtok != 5.0 {
		t.Errorf("failed to override with custom file, got %+v", p)
	}

	// Check new model from custom file
	p2 := Resolve("any", "new-model", nil)
	if p2 == nil || p2.InputPerMtok != 0.5 {
		t.Errorf("failed to load new model from custom file, got %+v", p2)
	}

	// Check that other embedded defaults still work
	p3 := Resolve("openai-chat", "deepseek-chat", nil)
	if p3 == nil || p3.InputPerMtok != 0.14 {
		t.Errorf("embedded default deepseek-chat lost after custom Init, got %+v", p3)
	}
}

func TestConcurrency(_ *testing.T) {
	_ = Init("")

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 100 {
				_ = Resolve("openai-chat", "gpt-4o", nil)
			}
		})
	}

	for range 10 {
		wg.Go(func() {
			_ = Init("")
		})
	}
	wg.Wait()
}
