// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/client-go/rest"

	seaplaneclientset "github.com/openshift-online/rosa-seaplane-api/clientset"
)

// DNSDomain is the OCM-compatible view of a Platform API DNS reservation.
// The Platform API has no dns_domains route: it exposes named DNS reservations,
// so ID is the reservation name and BaseDomain is the domain allocated in
// status once the reservation is Ready.
type DNSDomain struct {
	Kind                string    `json:"kind,omitempty"`
	ID                  string    `json:"id"`
	ClusterArch         string    `json:"cluster_arch"`
	UserDefined         bool      `json:"user_defined"`
	ReservedAtTimestamp time.Time `json:"reserved_at_timestamp"`
	BaseDomain          string    `json:"base_domain,omitempty"`
}

// dnsReservation is the subset of the Platform API DNSReservation resource that
// the CLI reads.
type dnsReservation struct {
	Metadata struct {
		Name              string    `json:"name"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
	} `json:"metadata"`
	Status struct {
		Phase      string `json:"phase"`
		BaseDomain string `json:"baseDomain"`
	} `json:"status"`
}

func (r dnsReservation) toDNSDomain() DNSDomain {
	return DNSDomain{
		Kind:                "DNSDomain",
		ID:                  r.Metadata.Name,
		ClusterArch:         "hcp",
		UserDefined:         true,
		ReservedAtTimestamp: r.Metadata.CreationTimestamp,
		BaseDomain:          r.Status.BaseDomain,
	}
}

// newDNSReservationName returns a reservation name the Platform API accepts as
// a DNS subdomain.
func newDNSReservationName() (string, error) {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("generate DNS reservation name: %w", err)
	}
	return fmt.Sprintf("rosa-%x", suffix[:]), nil
}

// DNSDomainClient reuses the SDK's authenticated transport for the native
// DNS-domain endpoints, which are not yet represented in the generated SDK.
type DNSDomainClient struct {
	client rest.Interface
}

func NewDNSDomainClient(client seaplaneclientset.Interface) *DNSDomainClient {
	return &DNSDomainClient{client: client.SeaplaneV1alpha1().RESTClient()}
}

func (c *DNSDomainClient) Create(ctx context.Context) (*DNSDomain, error) {
	name, err := newDNSReservationName()
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]any{"metadata": map[string]string{"name": name}})
	if err != nil {
		return nil, fmt.Errorf("encode DNS reservation request: %w", err)
	}
	result := c.client.Post().Resource("dns_reservations").SetHeader("Content-Type", "application/json").
		Body(body).Do(ctx)
	if err := result.Error(); err != nil {
		return nil, fmt.Errorf("create DNS domain: %w", err)
	}
	data, _ := result.Raw()
	var reservation dnsReservation
	if err := json.Unmarshal(data, &reservation); err != nil {
		return nil, fmt.Errorf("decode DNS domain: %w", err)
	}
	if reservation.Metadata.Name == "" {
		return nil, fmt.Errorf("DNS domain response has no ID")
	}
	domain := reservation.toDNSDomain()
	return &domain, nil
}

// List returns account-scoped HCP reservations. Platform API --all cannot
// expose other accounts or Classic domains.
func (c *DNSDomainClient) List(ctx context.Context) ([]DNSDomain, error) {
	result := c.client.Get().Resource("dns_reservations").Do(ctx)
	if err := result.Error(); err != nil {
		return nil, fmt.Errorf("list DNS domains: %w", err)
	}
	data, _ := result.Raw()
	var response struct {
		Items []dnsReservation `json:"items"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("decode DNS domain list: %w", err)
	}
	domains := make([]DNSDomain, 0, len(response.Items))
	for _, reservation := range response.Items {
		domains = append(domains, reservation.toDNSDomain())
	}
	return domains, nil
}

func (c *DNSDomainClient) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("DNS domain ID is required")
	}
	if err := c.client.Delete().Resource("dns_reservations").Name(id).Do(ctx).Error(); err != nil {
		return fmt.Errorf("delete DNS domain %q: %w", id, err)
	}
	return nil
}
