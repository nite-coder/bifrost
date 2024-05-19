package gateway

import "fmt"

type Bifrost struct {
	httpServers []*HTTPServer
}

func (b *Bifrost) Run() {
	for i := 0; i < len(b.httpServers)-1; i++ {
		go b.httpServers[i].Run()
	}

	b.httpServers[len(b.httpServers)-1].Run()
}

func Load(opts Options) (*Bifrost, error) {

	bifrsot := &Bifrost{}

	httpServers := map[string]*HTTPServer{}
	for _, entry := range opts.Entries {

		if entry.ID == "" {
			return nil, fmt.Errorf("http server id can't be empty")
		}

		if entry.Bind == "" {
			return nil, fmt.Errorf("http server bind can't be empty")
		}

		_, found := httpServers[entry.ID]
		if found {
			return nil, fmt.Errorf("http server '%s' already exists", entry.ID)
		}

		httpServer, err := NewHTTPServer(entry, opts)
		if err != nil {
			return nil, err
		}
		bifrsot.httpServers = append(bifrsot.httpServers, httpServer)
	}

	return bifrsot, nil
}

func LoadFromConfig(path string) (*Bifrost, error) {
	return nil, nil
}
