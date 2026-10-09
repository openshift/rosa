package hyperfleet

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/client-go/rest"

	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
)

// DNSDomain matches the Platform API's OCM-compatible DNS reservation response.
type DNSDomain struct {
	Kind                string    `json:"kind,omitempty"`
	ID                  string    `json:"id"`
	ClusterArch         string    `json:"cluster_arch"`
	UserDefined         bool      `json:"user_defined"`
	ReservedAtTimestamp time.Time `json:"reserved_at_timestamp"`
	Cluster             *struct {
		ID string `json:"id"`
	} `json:"cluster,omitempty"`
}

// DNSDomainClient reuses the SDK's authenticated transport for the native
// DNS-domain endpoints, which are not yet represented in the generated SDK.
type DNSDomainClient struct {
	client rest.Interface
}

func NewDNSDomainClient(client hyperfleetclientset.Interface) *DNSDomainClient {
	return &DNSDomainClient{client: client.HyperfleetV1alpha1().RESTClient()}
}

func (c *DNSDomainClient) Create(ctx context.Context) (*DNSDomain, error) {
	result := c.client.Post().Resource("dns_domains").SetHeader("Content-Type", "application/json").
		Body([]byte(`{"cluster_arch":"hcp"}`)).Do(ctx)
	if err := result.Error(); err != nil {
		return nil, fmt.Errorf("create DNS domain: %w", err)
	}
	data, _ := result.Raw()
	var domain DNSDomain
	if err := json.Unmarshal(data, &domain); err != nil {
		return nil, fmt.Errorf("decode DNS domain: %w", err)
	}
	if domain.ID == "" {
		return nil, fmt.Errorf("DNS domain response has no ID")
	}
	return &domain, nil
}

// List returns account-scoped HCP reservations. Platform API --all cannot
// expose other accounts or Classic domains.
func (c *DNSDomainClient) List(ctx context.Context) ([]DNSDomain, error) {
	result := c.client.Get().Resource("dns_domains").Do(ctx)
	if err := result.Error(); err != nil {
		return nil, fmt.Errorf("list DNS domains: %w", err)
	}
	data, _ := result.Raw()
	var response struct {
		Items []DNSDomain `json:"items"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("decode DNS domain list: %w", err)
	}
	if response.Items == nil {
		response.Items = []DNSDomain{}
	}
	return response.Items, nil
}

func (c *DNSDomainClient) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("DNS domain ID is required")
	}
	if err := c.client.Delete().Resource("dns_domains").Name(id).Do(ctx).Error(); err != nil {
		return fmt.Errorf("delete DNS domain %q: %w", id, err)
	}
	return nil
}
