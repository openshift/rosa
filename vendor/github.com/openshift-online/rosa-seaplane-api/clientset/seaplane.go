/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package seaplane provides a typed client for the Seaplane platform API.
//
// Usage:
//
//	awsCfg, _ := config.LoadDefaultConfig(ctx)
//	cs, err := seaplane.NewForConfig(&rest.Config{
//	    Host:      "https://abc123.execute-api.us-east-1.amazonaws.com",
//	    AccountID: "123456789012",
//	    AWSConfig: awsCfg,
//	})
//	cluster, err := cs.SeaplaneV1alpha1().Clusters().Get(ctx, "my-cluster", platform.GetOptions{})
package seaplane

import (
	"fmt"
	"net/http"

	generatedclientset "github.com/openshift-online/rosa-seaplane-api/clientset/generated"
	"github.com/openshift-online/rosa-seaplane-api/clientset/generated/scheme"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	hfrest "github.com/openshift-online/rosa-seaplane-api/clientset/rest"
	"github.com/openshift-online/rosa-seaplane-api/clientset/transport"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8srest "k8s.io/client-go/rest"
)

// Interface is the top-level client interface for the Seaplane platform API.
type Interface interface {
	SeaplaneV1alpha1() platform.V1alpha1PublicInterface
}

// Clientset implements Interface.
type Clientset struct {
	generated *generatedclientset.Clientset
}

// SeaplaneV1alpha1 returns the typed client for the seaplane.io/v1alpha1 group.
// Watch is disabled (returns ErrWatchNotSupported); use WaitUntil for polling-based waits.
func (c *Clientset) SeaplaneV1alpha1() platform.V1alpha1PublicInterface {
	return platform.NewV1alpha1PublicClient(c.generated.V1alpha1Public())
}

// NewForConfig creates a Clientset from a Config, wiring AWS SigV4 authentication
// and pointing the underlying REST client at /api/v0.
func NewForConfig(cfg *hfrest.Config) (*Clientset, error) {
	if cfg == nil {
		return nil, fmt.Errorf("seaplane: Config must not be nil")
	}
	if cfg.Host == "" {
		return nil, fmt.Errorf("seaplane: Config.Host is required")
	}
	if cfg.AccountID == "" {
		return nil, fmt.Errorf("seaplane: Config.AccountID is required")
	}
	if cfg.AWSConfig.Credentials == nil {
		return nil, fmt.Errorf("seaplane: Config.AWSConfig.Credentials must not be nil; use awsconfig.LoadDefaultConfig to populate it")
	}

	region, err := cfg.ResolveRegion()
	if err != nil {
		return nil, fmt.Errorf("seaplane: %w", err)
	}

	sigv4 := transport.New(nil, cfg.AWSConfig, region, cfg.AccountID, cfg.CallerARN)

	// Splitting the base path into APIPath="/api" + GroupVersion.Version="v0"
	// produces the same /api/v0 URL prefix while satisfying the scheme's
	// requirement that Version is non-empty for type registration.
	apiGV := schema.GroupVersion{Version: "v0"}
	metav1.AddToGroupVersion(scheme.Scheme, apiGV)
	restCfg := &k8srest.Config{
		Host:    cfg.Host,
		APIPath: "/api",
		ContentConfig: k8srest.ContentConfig{
			GroupVersion:         &apiGV,
			NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		},
		Transport: sigv4,
	}

	// Use RESTClientForConfigAndClient + New() rather than the generated
	// NewForConfigAndClient, which calls setConfigDefaults() and would
	// override our APIPath and GroupVersion back to Kubernetes API defaults.
	// The Adapter wraps sigv4 so response rewriting is decoupled from signing.
	httpClient := &http.Client{Transport: transport.NewAdapter(sigv4)}
	restClient, err := k8srest.RESTClientForConfigAndClient(restCfg, httpClient)
	if err != nil {
		return nil, fmt.Errorf("seaplane: building REST client: %w", err)
	}

	return &Clientset{generated: generatedclientset.New(restClient)}, nil
}
