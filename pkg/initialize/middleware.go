package initialize

import (
	_ "github.com/nite-coder/bifrost/pkg/middleware/addprefix"
	_ "github.com/nite-coder/bifrost/pkg/middleware/mirror"
	_ "github.com/nite-coder/bifrost/pkg/middleware/ratelimiting"
	_ "github.com/nite-coder/bifrost/pkg/middleware/replacepath"
	_ "github.com/nite-coder/bifrost/pkg/middleware/replacepathregex"
	_ "github.com/nite-coder/bifrost/pkg/middleware/requesttermination"
	_ "github.com/nite-coder/bifrost/pkg/middleware/requesttransformer"
	_ "github.com/nite-coder/bifrost/pkg/middleware/setvars"
	_ "github.com/nite-coder/bifrost/pkg/middleware/stripprefix"
	_ "github.com/nite-coder/bifrost/pkg/middleware/timinglogger"
	_ "github.com/nite-coder/bifrost/pkg/middleware/tracing"
	_ "github.com/nite-coder/bifrost/pkg/middleware/trafficsplitter"
)

func Middleware() error {
	return nil
}
