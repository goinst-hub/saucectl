package http

import (
	"context"
	"io"
	"net/http"

	"github.com/saucelabs/saucectl/internal/version"
)

// NewRequestWithContext is a wrapper around http.NewRequestWithContext that modifies the request by adding additional
// headers.
func NewRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	r, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return r, err
	}
	r.Header.Set("User-Agent", "saucectl/"+version.Version)

	return r, err
}
