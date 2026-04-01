package ratelimit

import (
	"sync"
	"time"
)

// allowResultPool is a sync.Pool for AllowResult objects.
var allowResultPool = sync.Pool{
	New: func() any {
		return &AllowResult{}
	},
}

// GetAllowResult gets an AllowResult from the pool.
func GetAllowResult() *AllowResult {
	res, ok := allowResultPool.Get().(*AllowResult)
	if !ok {
		return &AllowResult{}
	}
	return res
}

// PutAllowResult returns an AllowResult to the pool.
func PutAllowResult(result *AllowResult) {
	// Reset the object before putting it back
	result.Allow = false
	result.Limit = 0
	result.Remaining = 0
	result.ResetTime = time.Time{}
	allowResultPool.Put(result)
}
