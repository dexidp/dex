package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dexidp/dex/storage"

	"github.com/dexidp/dex/middleware"
	"github.com/dexidp/dex/middleware/groups"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

// Middleware is a middleware with resource version metadata.
type Middleware struct {
	ResourceVersion string
	Middleware      middleware.Middleware
}

// MiddlewareConfig is a configuration that can open a middleware.
type MiddlewareConfig interface {
	Open(logger log.Logger) (middleware.Middleware, error)
}

// MiddlewaresConfig variable provides an easy way to return a config struct
// depending on the middleware type.
var MiddlewaresConfig = map[string]func() MiddlewareConfig{
	"groups": func() MiddlewareConfig { return new(groups.Config) },
}

// openMiddleware will parse the middleware config and open the middleware.
func openMiddleware(logger log.Logger, mware storage.Middleware) (middleware.Middleware, error) {
	var m middleware.Middleware

	f, ok := MiddlewaresConfig[mware.Type]
	if !ok {
		return m, fmt.Errorf("unknown middleware type %q", mware.Type)
	}

	mwareConfig := f()
	if len(mware.Config) != 0 {
		if err := json.Unmarshal(mware.Config, mwareConfig); err != nil {
			return m, fmt.Errorf("parse middleware config: %v", err)
		}
	}

	m, err := mwareConfig.Open(logger)
	if err != nil {
		return m, fmt.Errorf("failed to create middleware %q: %v", mware.Type, err)
	}

	return m, nil
}

func (s *Server) OpenMiddleware(mware storage.Middleware) (Middleware, error) {
	m, err := openMiddleware(s.logger, mware)
	if err != nil {
		return Middleware{}, fmt.Errorf("failed to open middleware: %v", err)
	}

	middleware := Middleware{
		ResourceVersion: mware.ResourceVersion,
		Middleware:      m,
	}

	s.mu.Lock()
	s.middleware = append(s.middleware, middleware)
	s.mu.Unlock()

	return middleware, nil
}

// RunMiddleware executes the middleware for the specified connector, followed
// by the global middleware.
func (s *Server) RunMiddleware(ctx context.Context, conn Connector, identity connector.Identity) (connector.Identity, error) {
	var err error

	// First, run the connector middleware
	for _, mware := range conn.Middleware {
		identity, err = mware.Process(ctx, identity)
		if err != nil {
			return identity, err
		}
	}

	// Grab a copy of the global middleware
	s.mu.Lock()
	middleware := s.middleware
	s.mu.Unlock()

	for _, mware := range middleware {
		identity, err = mware.Middleware.Process(ctx, identity)
		if err != nil {
			return identity, err
		}
	}

	return identity, err
}
