package initialize

import (
	// register middleware
	_ "github.com/nite-coder/bifrost/pkg/middleware/addprefix"
	_ "github.com/nite-coder/bifrost/pkg/middleware/compression"
	_ "github.com/nite-coder/bifrost/pkg/middleware/coraza"
	_ "github.com/nite-coder/bifrost/pkg/middleware/cors"
	_ "github.com/nite-coder/bifrost/pkg/middleware/iprestriction"
	_ "github.com/nite-coder/bifrost/pkg/middleware/mirror"
	_ "github.com/nite-coder/bifrost/pkg/middleware/ratelimit"
	_ "github.com/nite-coder/bifrost/pkg/middleware/replacepath"
	_ "github.com/nite-coder/bifrost/pkg/middleware/replacepathregex"
	_ "github.com/nite-coder/bifrost/pkg/middleware/requesttermination"
	_ "github.com/nite-coder/bifrost/pkg/middleware/requesttransformer"
	_ "github.com/nite-coder/bifrost/pkg/middleware/responsetransformer"
	_ "github.com/nite-coder/bifrost/pkg/middleware/setvars"
	_ "github.com/nite-coder/bifrost/pkg/middleware/stripprefix"
	_ "github.com/nite-coder/bifrost/pkg/middleware/trafficsplitter"
	_ "github.com/nite-coder/bifrost/pkg/middleware/uarestriction"

	// register balance
	_ "github.com/nite-coder/bifrost/pkg/balancer/hasing"
	_ "github.com/nite-coder/bifrost/pkg/balancer/random"
	_ "github.com/nite-coder/bifrost/pkg/balancer/roundrobin"
	_ "github.com/nite-coder/bifrost/pkg/balancer/weighted"
)


func Bifrost() error {
	return nil
}
