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

var _ = Describe("Delete cluster", func() {
	var (
		t         *test.TestingRuntime
		clusterId string
	)

	BeforeEach(func() {
		t = test.NewTestRuntime()
		clusterId = test.MockClusterID
	})

	Context("handleClusterDelete", func() {
		It("returns nil and logs info when the cluster is already uninstalling", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})
			t.SetCluster(clusterId, clusterReady)

			statusBody := fmt.Sprintf(`{
				"kind": "ClusterStatus",
				"id": "%s",
				"state": "uninstalling"
			}`, clusterId)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, statusBody))

			err := t.StdOutReader.Record()
			Expect(err).NotTo(HaveOccurred())

			err = handleClusterDelete(t.RosaRuntime, clusterReady, clusterId, false)
			Expect(err).NotTo(HaveOccurred())

			stdout, err := t.StdOutReader.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("already uninstalling"))
		})

		It("deletes the cluster and logs the start message", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})
			t.SetCluster(clusterId, clusterReady)

			statusBody := fmt.Sprintf(`{
				"kind": "ClusterStatus",
				"id": "%s",
				"state": "ready"
			}`, clusterId)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, statusBody))
			t.ApiServer.AppendHandlers(RespondWithJSON(
				http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{clusterReady})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))

			err := t.StdOutReader.Record()
			Expect(err).NotTo(HaveOccurred())

			err = handleClusterDelete(t.RosaRuntime, clusterReady, clusterId, false)
			Expect(err).NotTo(HaveOccurred())

			stdout, err := t.StdOutReader.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("will start uninstalling"))
		})

		It("passes the bestEffort flag through to DeleteCluster", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})
			t.SetCluster(clusterId, clusterReady)

			statusBody := fmt.Sprintf(`{
				"kind": "ClusterStatus",
				"id": "%s",
				"state": "ready"
			}`, clusterId)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, statusBody))
			t.ApiServer.AppendHandlers(RespondWithJSON(
				http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{clusterReady})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))

			err := handleClusterDelete(t.RosaRuntime, clusterReady, clusterId, true)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error when GetClusterState fails", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})
			t.SetCluster(clusterId, clusterReady)

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, ""))

			err := handleClusterDelete(t.RosaRuntime, clusterReady, clusterId, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected response content type"))
		})

		It("returns an error when DeleteCluster fails", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})
			t.SetCluster(clusterId, clusterReady)

			statusBody := fmt.Sprintf(`{
				"kind": "ClusterStatus",
				"id": "%s",
				"state": "ready"
			}`, clusterId)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, statusBody))
			t.ApiServer.AppendHandlers(RespondWithJSON(
				http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{clusterReady})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusForbidden, `{
				"kind": "Error",
				"id": "403",
				"href": "/api/clusters_mgmt/v1/errors/403",
				"code": "CLUSTERS-MGMT-403",
				"reason": "forbidden"
			}`))

			err := handleClusterDelete(t.RosaRuntime, clusterReady, clusterId, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("forbidden"))
		})
	})

	Context("buildCommands", func() {
		It("uses cluster ID flags when OIDC config is not reusable", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/Installer").
						OperatorRolePrefix("my-prefix").
						OIDCEndpointURL("https://oidc.example.com").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/op-role"),
						),
				))
			})

			result := buildCommands(cluster)
			Expect(result).To(ContainSubstring(fmt.Sprintf("-c %s", clusterId)))
			Expect(result).To(ContainSubstring("rosa delete operator-roles"))
			Expect(result).To(ContainSubstring("rosa delete oidc-provider"))
			Expect(result).NotTo(ContainSubstring("--prefix"))
			Expect(result).NotTo(ContainSubstring("--oidc-config-id"))
		})

		It("uses prefix and oidc-config-id flags when OIDC config is reusable", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/Installer").
						OperatorRolePrefix("my-prefix").
						OIDCEndpointURL("https://oidc.example.com").
						OidcConfig(cmv1.NewOidcConfig().ID("oidc-abc123").Reusable(true)).
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/op-role"),
						),
				))
			})

			result := buildCommands(cluster)
			Expect(result).To(ContainSubstring("--prefix my-prefix"))
			Expect(result).To(ContainSubstring("--oidc-config-id oidc-abc123"))
			Expect(result).NotTo(ContainSubstring(fmt.Sprintf("-c %s", clusterId)))
		})
	})
})
