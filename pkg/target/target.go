package target

// Target groups endpoints under a logical name with weight and tags.
type Target struct {
	Name      string
	Weight    uint32
	Tags      map[string]string
	Endpoints map[string]*Endpoint
}
