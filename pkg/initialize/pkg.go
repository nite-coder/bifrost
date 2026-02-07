package initialize

import (
	// register middleware
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

	// register balance
	"github.com/nite-coder/bifrost/pkg/balancer/chash"
	"github.com/nite-coder/bifrost/pkg/balancer/random"
	"github.com/nite-coder/bifrost/pkg/balancer/roundrobin"
	"github.com/nite-coder/bifrost/pkg/balancer/weighted"
)

func Bifrost() error {
	// middleware
	if err := addprefix.Init(); err != nil {
		return err
	}

	if err := buffering.Init(); err != nil {
		return err
	}

	if err := compression.Init(); err != nil {
		return err
	}

	if err := coraza.Init(); err != nil {
		return err
	}

	if err := cors.Init(); err != nil {
		return err
	}

	if err := iprestriction.Init(); err != nil {
		return err
	}

	if err := mirror.Init(); err != nil {
		return err
	}

	if err := parallel.Init(); err != nil {
		return err
	}

	if err := ratelimit.Init(); err != nil {
		return err
	}

	if err := replacepath.Init(); err != nil {
		return err
	}

	if err := replacepathregex.Init(); err != nil {
		return err
	}

	if err := requesttermination.Init(); err != nil {
		return err
	}

	if err := requesttransformer.Init(); err != nil {
		return err
	}

	if err := responsetransformer.Init(); err != nil {
		return err
	}

	if err := setvars.Init(); err != nil {
		return err
	}

	if err := stripprefix.Init(); err != nil {
		return err
	}

	if err := trafficsplitter.Init(); err != nil {
		return err
	}

	if err := uarestriction.Init(); err != nil {
		return err
	}

	// balancer
	if err := chash.Init(); err != nil {
		return err
	}

	if err := random.Init(); err != nil {
		return err
	}

	if err := roundrobin.Init(); err != nil {
		return err
	}

	if err := weighted.Init(); err != nil {
		return err
	}

	return nil
}
