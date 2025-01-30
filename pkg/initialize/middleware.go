package initialize

import (
	"testing"

	_ "github.com/nite-coder/bifrost/pkg/middleware/addprefix"
	_ "github.com/nite-coder/bifrost/pkg/middleware/mirror"
	_ "github.com/nite-coder/bifrost/pkg/middleware/opentelemetry"
	_ "github.com/nite-coder/bifrost/pkg/middleware/ratelimiting"
	_ "github.com/nite-coder/bifrost/pkg/middleware/replacepath"
	_ "github.com/nite-coder/bifrost/pkg/middleware/replacepathregex"
	_ "github.com/nite-coder/bifrost/pkg/middleware/requesttermination"
	_ "github.com/nite-coder/bifrost/pkg/middleware/requesttransformer"
	_ "github.com/nite-coder/bifrost/pkg/middleware/responsetransformer"
	_ "github.com/nite-coder/bifrost/pkg/middleware/setvars"
	_ "github.com/nite-coder/bifrost/pkg/middleware/stripprefix"
	_ "github.com/nite-coder/bifrost/pkg/middleware/trafficsplitter"
	"github.com/stretchr/testify/assert"
)

func Middleware() error {
	return nil
}

func TestInitMiddleware(t *testing.T) {
	err := Middleware()
	assert.NoError(t, err)
}
