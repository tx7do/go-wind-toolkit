package mux

import (
	"fmt"
	"io"
	"strings"

	"ariga.io/atlas/sql/schema"
)

type (
	convertProvider func(string) (*ConvertDriver, error)

	Mux struct {
		providers map[string]convertProvider
	}

	// ConvertDriver implements Inspector interface and holds inspection information.
	ConvertDriver struct {
		io.Closer
		schema.Inspector
		Dialect    string
		SchemaName string
	}
)

// New returns a new Mux.
func New() *Mux {
	return &Mux{
		providers: make(map[string]convertProvider),
	}
}

var Default = New()

// RegisterProvider is used to register an Atlas provider by key.
func (u *Mux) RegisterProvider(p convertProvider, scheme ...string) {
	for _, s := range scheme {
		u.providers[s] = p
	}
}

// OpenConvert is used for opening an import driver on a specific data source.
func (u *Mux) OpenConvert(dsn string) (*ConvertDriver, error) {
	scheme, host, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %v", err)
	}
	p, ok := u.providers[scheme]
	if !ok {
		return nil, fmt.Errorf("provider does not exist: %q", scheme)
	}
	return p(host)
}

func parseDSN(url string) (string, string, error) {
	a := strings.SplitN(url, "://", 2)
	if len(a) != 2 {
		return "", "", fmt.Errorf(`failed to parse dsn: "%s"`, url)
	}
	return a[0], a[1], nil
}
