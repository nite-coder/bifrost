package initialize

import (
	"github.com/nite-coder/bifrost/pkg/ai/pricing"
	"github.com/nite-coder/bifrost/pkg/config"
)

// AI initializes the AI pricing registry.
func AI(options config.Options) error {
	if options.AI != nil {
		err := pricing.Init(options.AI.PricingFile)
		if err != nil {
			return err
		}
	}
	return nil
}
