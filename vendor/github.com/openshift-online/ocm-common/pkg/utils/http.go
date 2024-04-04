package utils

import (
	"net/http"
)

//go:generate mockgen -source=http.go -package=clients -destination=./http_mock.go
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

var _ HTTPClient = &DefaultHTTPClient{}

// DefaultHTTPClient is a wrapper around http.Client that implements the HTTPClient interface.
type DefaultHTTPClient struct {
	client *http.Client
}

func NewDefaultHTTPClient(client *http.Client) HTTPClient {
	return &DefaultHTTPClient{
		client: client,
	}
}

func (c *DefaultHTTPClient) Get(url string) (*http.Response, error) {
	return c.client.Get(url)
}
