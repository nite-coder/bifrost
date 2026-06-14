package target

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// EndpointHash returns a deterministic hash of the given endpoints.
func EndpointHash(endpoints []*Endpoint) string {
	if len(endpoints) == 0 {
		return ""
	}

	sorted := make([]*Endpoint, 0, len(endpoints))
	for _, ep := range endpoints {
		if ep != nil {
			sorted = append(sorted, ep)
		}
	}
	if len(sorted) == 0 {
		return ""
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Address < sorted[j].Address
	})

	h := sha256.New()
	for _, ep := range sorted {
		_, _ = h.Write([]byte(ep.Address))
		_, _ = fmt.Fprintf(h, "%d", ep.Weight)
		if len(ep.Tags) > 0 {
			tagKeys := make([]string, 0, len(ep.Tags))
			for k := range ep.Tags {
				tagKeys = append(tagKeys, k)
			}
			sort.Strings(tagKeys)
			for _, k := range tagKeys {
				_, _ = h.Write([]byte(k))
				_, _ = h.Write([]byte(ep.Tags[k]))
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
