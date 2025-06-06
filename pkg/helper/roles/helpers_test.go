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

package roles

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/test"
)

func TestRolesHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Roles Helper Suite")
}

var _ = Describe("Roles Helper", func() {
	Context("Validate Additional Allowed Principals", func() {
		It("should pass when valid ARNs", func() {
			aapARNs := []string{
				"arn:aws:iam::123456789012:role/role1",
				"arn:aws:iam::123456789012:role/role2",
			}
			err := ValidateAdditionalAllowedPrincipals(aapARNs)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("should error when containing duplicate ARNs", func() {
			aapARNs := []string{
				"arn:aws:iam::123456789012:role/role1",
				"arn:aws:iam::123456789012:role/role2",
				"arn:aws:iam::123456789012:role/role1",
			}
			err := ValidateAdditionalAllowedPrincipals(aapARNs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"Invalid additional allowed principals list, " +
					"duplicate key 'arn:aws:iam::123456789012:role/role1' found"))
		})

		It("should error when contain invalid ARN", func() {
			aapARNs := []string{
				"arn:aws:iam::123456789012:role/role1",
				"foobar",
			}
			err := ValidateAdditionalAllowedPrincipals(aapARNs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Expected valid ARNs for additional allowed principals list"))
		})

	})
	Context("ValidateAccountAndOperatorRolesManagedPolicies", func() {
		var cluster *cmv1.Cluster
		var credRequest map[string]*cmv1.STSOperator
		var version4130 *cmv1.VersionBuilder
		var version4130WithUpgrades *cmv1.VersionBuilder
		var t test.TestingRuntime
		var mockAWS *mock.MockClient
		var policiesResponse string
		var opPoliciesResponse string

		BeforeEach(func() {
			t.InitRuntime()

			mockCtrl := gomock.NewController(GinkgoT())
			mockAWS = mock.NewMockClient(mockCtrl)
			t.RosaRuntime.AWSClient = mockAWS

			stsOperator1, err := cmv1.NewSTSOperator().Namespace("namespace-1").Build()
			Expect(err).NotTo(HaveOccurred())
			stsOperator2, err := cmv1.NewSTSOperator().Namespace("namespace-2").Build()
			Expect(err).NotTo(HaveOccurred())
			credRequest = map[string]*cmv1.STSOperator{
				"operator-1": stsOperator1,
				"operator-2": stsOperator2,
			}

			version4130 = cmv1.NewVersion().ID("openshift-v4.13.0").RawID("4.13.0").
				ReleaseImage("1").HREF("/api/clusters_mgmt/v1/versions/openshift-v4.13.0").
				Enabled(true).ChannelGroup("stable").ROSAEnabled(true).
				HostedControlPlaneEnabled(true)

			version4130WithUpgrades = version4130.AvailableUpgrades("4.13.1")

			awsBuilder := cmv1.NewAWS().STS(cmv1.NewSTS().ManagedPolicies(true))

			cluster = test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
				c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
				c.State(cmv1.ClusterStateReady)
				c.Hypershift(cmv1.NewHypershift().Enabled(true))
				c.AWS(awsBuilder)
				c.Version(version4130WithUpgrades)
			})

			policy, _ := cmv1.NewAWSSTSPolicy().ID("123").Type("").Build()
			policies := make([]*cmv1.AWSSTSPolicy, 0)
			policies = append(policies, policy)
			policiesResponse = test.FormatAWSSTSPolicyList(policies)

			opPolicy, _ := cmv1.NewAWSSTSPolicy().ID("123").Type("OperatorRole").Build()
			opPolicies := make([]*cmv1.AWSSTSPolicy, 0)
			opPolicies = append(opPolicies, opPolicy)
			opPoliciesResponse = test.FormatAWSSTSPolicyList(opPolicies)

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, policiesResponse))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, opPoliciesResponse))
		})

		It("Managed policies are validated successfully", func() {
			mockAWS.EXPECT().ValidateHCPAccountRolesManagedPolicies("", gomock.Any()).Return(nil)
			mockAWS.EXPECT().ValidateOperatorRolesManagedPolicies(
				cluster, gomock.Any(), gomock.Any(), gomock.Any(),
			).Return(nil)

			err := ValidateAccountAndOperatorRolesManagedPolicies(
				t.RosaRuntime,
				cluster,
				credRequest,
				"",
				"auto",
				cluster.Version().AvailableUpgrades()[0])
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("Managed policies are validated successfully without cluster upgrade version", func() {
			mockAWS.EXPECT().ValidateHCPAccountRolesManagedPolicies("", gomock.Any()).Return(nil)
			mockAWS.EXPECT().ValidateOperatorRolesManagedPolicies(
				cluster, gomock.Any(), gomock.Any(), gomock.Any(),
			).Return(nil)

			err := ValidateAccountAndOperatorRolesManagedPolicies(
				t.RosaRuntime,
				cluster,
				credRequest,
				"",
				"auto",
				"")
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("Account policies failures cause validation to err", func() {
			mockAWS.EXPECT().ValidateHCPAccountRolesManagedPolicies(
				"", gomock.Any(),
			).Return(fmt.Errorf("could not find"))

			err := ValidateAccountAndOperatorRolesManagedPolicies(
				t.RosaRuntime,
				cluster,
				credRequest,
				"",
				"auto",
				cluster.Version().AvailableUpgrades()[0])
			Expect(err).Should(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("could not find")))
		})

		It("Operator policies failures cause validation to err", func() {
			mockAWS.EXPECT().ValidateHCPAccountRolesManagedPolicies("", gomock.Any()).Return(nil)
			mockAWS.EXPECT().ValidateOperatorRolesManagedPolicies(
				cluster, gomock.Any(), gomock.Any(), gomock.Any(),
			).Return(fmt.Errorf("could not find"))

			err := ValidateAccountAndOperatorRolesManagedPolicies(
				t.RosaRuntime,
				cluster,
				credRequest,
				"",
				"auto",
				cluster.Version().AvailableUpgrades()[0])
			Expect(err).Should(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("could not find")))
		})
	})
})
