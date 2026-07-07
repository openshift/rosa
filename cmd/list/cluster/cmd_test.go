package cluster

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	awsClient "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("List clusters", func() {
	Context("clusterTopology", func() {
		It("returns Classic for a cluster without STS or Hypershift", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.Hypershift(cmv1.NewHypershift().Enabled(false))
			})
			Expect(clusterTopology(cluster)).To(Equal("Classic"))
		})

		It("returns Classic (STS) when AWS STS is enabled", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS().Enabled(true).RoleARN("arn:aws:iam::123:role/Installer")))
				c.Hypershift(cmv1.NewHypershift().Enabled(false))
			})
			Expect(clusterTopology(cluster)).To(Equal("Classic (STS)"))
		})

		It("returns Hosted CP when Hypershift is enabled", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.Hypershift(cmv1.NewHypershift().Enabled(true))
			})
			Expect(clusterTopology(cluster)).To(Equal("Hosted CP"))
		})

		It("returns Hosted CP when both Hypershift and STS are enabled", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS().Enabled(true).RoleARN("arn:aws:iam::123:role/Installer")))
				c.Hypershift(cmv1.NewHypershift().Enabled(true))
			})
			Expect(clusterTopology(cluster)).To(Equal("Hosted CP"))
		})
	})

	Context("listClustersUsingAccountRole", func() {
		var t *test.TestingRuntime

		BeforeEach(func() {
			t = test.NewTestRuntime()
		})

		It("returns clusters when AWS role lookup and OCM query succeed", func() {
			args.accountRoleArn = "arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role"

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().GetAccountRoleByArn(args.accountRoleArn).Return(awsClient.Role{
				RoleName: "ManagedOpenShift-Installer-Role",
				RoleARN:  args.accountRoleArn,
				RoleType: "Installer",
			}, nil)

			mockCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})
			t.ApiServer.AppendHandlers(RespondWithJSON(
				http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{mockCluster})))

			clusters, err := listClustersUsingAccountRole(t.RosaRuntime.Creator, t.RosaRuntime)
			Expect(err).NotTo(HaveOccurred())
			Expect(clusters).To(HaveLen(1))
			Expect(clusters[0].ID()).To(Equal(test.MockClusterID))
		})

		It("returns error when GetAccountRoleByArn fails", func() {
			args.accountRoleArn = "arn:aws:iam::123456789012:role/NonExistent"

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().GetAccountRoleByArn(args.accountRoleArn).Return(
				awsClient.Role{}, fmt.Errorf("role not found"))

			_, err := listClustersUsingAccountRole(t.RosaRuntime.Creator, t.RosaRuntime)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("role not found"))
		})

		It("returns error when GetClustersUsingAccountRole fails", func() {
			args.accountRoleArn = "arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role"

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().GetAccountRoleByArn(args.accountRoleArn).Return(awsClient.Role{
				RoleName: "ManagedOpenShift-Installer-Role",
				RoleARN:  args.accountRoleArn,
				RoleType: "Installer",
			}, nil)

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, `{
				"kind": "Error",
				"id": "500",
				"href": "/api/clusters_mgmt/v1/errors/500",
				"code": "CLUSTERS-MGMT-500",
				"reason": "internal error"
			}`))

			_, err := listClustersUsingAccountRole(t.RosaRuntime.Creator, t.RosaRuntime)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected response content type"))
		})
	})
})
