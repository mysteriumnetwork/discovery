package pricing

import (
	"context"
	"net/http"
	"net/url"

	"github.com/prometheus/client_golang/api"
	promV1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// NewPromClient creates new Prometheus API client.
func NewPromClient(address, user, password string) promV1.API {
	c, err := api.NewClient(api.Config{
		Address:      address,
		RoundTripper: api.DefaultRoundTripper,
	})
	if err != nil {
		panic(err)
	}

	return promV1.NewAPI(newAPIClientWithBasicAuth(c, user, password))
}

func newAPIClientWithBasicAuth(c api.Client, user, password string) api.Client {
	return &apiClientWithBasicAuth{
		client:   c,
		user:     user,
		password: password,
	}
}

type apiClientWithBasicAuth struct {
	client   api.Client
	user     string
	password string
}

func (c *apiClientWithBasicAuth) URL(ep string, args map[string]string) *url.URL {
	return c.client.URL(ep, args)
}

func (c *apiClientWithBasicAuth) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	req.SetBasicAuth(c.user, c.password)
	return c.client.Do(ctx, req)
}
