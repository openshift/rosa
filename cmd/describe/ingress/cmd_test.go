/*
Copyright (c) 2024 Red Hat, Inc.

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

package ingress

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Get min width for output", func() {
	It("retrieves the min width", func() {
		minWidth := getMinWidth([]string{"a", "ab", "abc", "def"})
		Expect(minWidth).To(Equal(3))
	})
	When("empty slice", func() {
		It("retrieves the min width as 0", func() {
			minWidth := getMinWidth([]string{})
			Expect(minWidth).To(Equal(0))
		})
	})
})

var _ = Describe("Retrieve map of entries for output", func() {
	It("retrieves map", func() {
		cluster, err := v1.NewCluster().ID("123").Build()
		Expect(err).To(BeNil())
		ingress, err := v1.NewIngress().
			ID("123").
			Default(true).
			Listening(v1.ListeningMethodExternal).
			LoadBalancerType(v1.LoadBalancerFlavorNlb).
			RouteWildcardPolicy(v1.WildcardPolicyWildcardsAllowed).
			RouteNamespaceOwnershipPolicy(v1.NamespaceOwnershipPolicyStrict).
			RouteSelectors(map[string]string{
				"test-route": "test-selector",
			}).
			ExcludedNamespaces("test", "test2").
			ComponentRoutes(map[string]*v1.ComponentRouteBuilder{
				string(v1.ComponentRouteTypeOauth): v1.NewComponentRoute().
					Hostname("oauth-hostname").TlsSecretRef("oauth-secret"),
			}).
			Build()
		Expect(err).To(BeNil())
		mapOutput := generateEntriesOutput(cluster, ingress)
		Expect(mapOutput).To(HaveLen(10))
	})
})
