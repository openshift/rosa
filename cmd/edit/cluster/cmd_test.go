/*
Copyright (c) 2023 Red Hat, Inc.

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

package cluster

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/openshift/rosa/pkg/test"
)

const (
	clusterName                  = "fakeClusterName"
	existingPrivateWarningString = "warning string "
)

var _ = Describe("Edit cluster", func() {
	Context("warnUserForOAuthHCPVisibility", func() {
		var testRuntime test.TestingRuntime
		mockHypershiftClusterReady, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})
		Expect(err).To(BeNil())

		mockClassicCluster, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(false))
		})
		Expect(err).To(BeNil())

		BeforeEach(func() {
			testRuntime.InitRuntime()
		})
		It("Return input string for classic cluster", func() {
			outputString, err := warnUserForOAuthHCPVisibility(testRuntime.RosaRuntime,
				clusterName, mockClassicCluster, existingPrivateWarningString)
			Expect(err).To(BeNil())
			Expect(outputString).To(Equal(existingPrivateWarningString))
		})
		It("Return error if ingress call fails", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, ""))
			outputString, err := warnUserForOAuthHCPVisibility(testRuntime.RosaRuntime,
				clusterName, mockHypershiftClusterReady, existingPrivateWarningString)
			Expect(err).To(Not(BeNil()))
			Expect(outputString).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("failed to get ingresses for cluster '%s", clusterName)))
		})
		It("Return input string for HyperShift cluster with no ingress", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatIngressList(buildTestIngresses(0, 0))))
			outputString, err := warnUserForOAuthHCPVisibility(testRuntime.RosaRuntime,
				clusterName, mockHypershiftClusterReady, existingPrivateWarningString)
			Expect(err).To(BeNil())
			Expect(outputString).To(Equal(existingPrivateWarningString))
		})
		It("Return input string for  HyperShift cluster with no public ingress", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatIngressList(buildTestIngresses(3, 0))))
			outputString, err := warnUserForOAuthHCPVisibility(testRuntime.RosaRuntime,
				clusterName, mockHypershiftClusterReady, existingPrivateWarningString)
			Expect(err).To(BeNil())
			Expect(outputString).To(Equal(existingPrivateWarningString))
		})
		It("Append string for HyperShift cluster with some public ingress", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatIngressList(buildTestIngresses(3, 2))))
			outputString, err := warnUserForOAuthHCPVisibility(testRuntime.RosaRuntime,
				clusterName, mockHypershiftClusterReady, existingPrivateWarningString)
			Expect(err).To(BeNil())
			Expect(outputString).To(
				ContainSubstring("warning string OAuth visibility will be affected by cluster visibility change"))
		})
	})
})

func buildTestIngresses(total int, public int) []*cmv1.Ingress {
	Expect(public).Should(BeNumerically("<=", total))
	ingresses := make([]*cmv1.Ingress, 0)
	currentPublic := 0
	for i := 0; i < total; i++ {
		ingressBuilder := cmv1.NewIngress().ID(fmt.Sprintf("ingress%d", i))
		if public > currentPublic {
			ingressBuilder.Listening(cmv1.ListeningMethodExternal)
			currentPublic += 1
		} else {
			ingressBuilder.Listening(cmv1.ListeningMethodInternal)
		}
		ingress, err := ingressBuilder.Build()
		Expect(err).To(BeNil())
		ingresses = append(ingresses, ingress)
	}
	return ingresses
}
