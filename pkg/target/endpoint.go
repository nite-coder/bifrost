package target

// Endpoint represents an upstream endpoint address with weight and state.
type Endpoint struct {
	Address string
	Weight  uint32
	Tags    map[string]string
	State   *State
}
