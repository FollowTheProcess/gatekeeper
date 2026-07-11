package auth

import (
	"context"
)

// Fetcher is a project-specific public-key fetcher.
type Fetcher interface {
	// Fetch fetches a project-specific public key from some data store, returning
	// the raw PEM bytes.
	Fetch(ctx context.Context, project string) ([]byte, error)
}
