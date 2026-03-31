package initialize

import (
	"github.com/nite-coder/bifrost/pkg/balancer/chash"
	"github.com/nite-coder/bifrost/pkg/balancer/random"
	"github.com/nite-coder/bifrost/pkg/balancer/roundrobin"
	"github.com/nite-coder/bifrost/pkg/balancer/weighted"
	"github.com/nite-coder/bifrost/pkg/middleware/addprefix"
	"github.com/nite-coder/bifrost/pkg/middleware/buffering"
	"github.com/nite-coder/bifrost/pkg/middleware/compression"
	"github.com/nite-coder/bifrost/pkg/middleware/coraza"
	"github.com/nite-coder/bifrost/pkg/middleware/cors"
	"github.com/nite-coder/bifrost/pkg/middleware/iprestriction"
	"github.com/nite-coder/bifrost/pkg/middleware/mirror"
	"github.com/nite-coder/bifrost/pkg/middleware/parallel"
	"github.com/nite-coder/bifrost/pkg/middleware/ratelimit"
	"github.com/nite-coder/bifrost/pkg/middleware/replacepath"
	"github.com/nite-coder/bifrost/pkg/middleware/replacepathregex"
	"github.com/nite-coder/bifrost/pkg/middleware/requesttermination"
	"github.com/nite-coder/bifrost/pkg/middleware/requesttransformer"
	"github.com/nite-coder/bifrost/pkg/middleware/responsetransformer"
	"github.com/nite-coder/bifrost/pkg/middleware/setvars"
	"github.com/nite-coder/bifrost/pkg/middleware/stripprefix"
	"github.com/nite-coder/bifrost/pkg/middleware/trafficsplitter"
	"github.com/nite-coder/bifrost/pkg/middleware/uarestriction"
)

// Bifrost initializes all standard middlewares and balancers.
func Bifrost() error {
	// middleware
	err := addprefix.Init()
	if err != nil {
		return err
	}

	err = buffering.Init()
	if err != nil {
		return err
	}

	err = compression.Init()
	if err != nil {
		return err
	}

	err = coraza.Init()
	if err != nil {
		return err
	}

	err = cors.Init()
	if err != nil {
		return err
	}

	err = iprestriction.Init()
	if err != nil {
		return err
	}

	err = mirror.Init()
	if err != nil {
		return err
	}

	err = parallel.Init()
	if err != nil {
		return err
	}

	err = ratelimit.Init()
	if err != nil {
		return err
	}

	err = replacepath.Init()
	if err != nil {
		return err
	}

	err = replacepathregex.Init()
	if err != nil {
		return err
	}

	err = requesttermination.Init()
	if err != nil {
		return err
	}

	err = requesttransformer.Init()
	if err != nil {
		return err
	}

	err = responsetransformer.Init()
	if err != nil {
		return err
	}

	err = setvars.Init()
	if err != nil {
		return err
	}

	err = stripprefix.Init()
	if err != nil {
		return err
	}

	err = trafficsplitter.Init()
	if err != nil {
		return err
	}

	err = uarestriction.Init()
	if err != nil {
		return err
	}

	// balancer
	err = chash.Init()
	if err != nil {
		return err
	}

	err = random.Init()
	if err != nil {
		return err
	}

	err = roundrobin.Init()
	if err != nil {
		return err
	}

	err = weighted.Init()
	if err != nil {
		return err
	}

	return nil
}
