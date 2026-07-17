package cluster

import (
	"fmt"
	"io"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/test"
)

func captureRun(fn func() error) (string, string, error) {
	rout, wout, _ := os.Pipe()
	rerr, werr, _ := os.Pipe()
	origOut := os.Stdout
	origErr := os.Stderr
	defer func() {
		os.Stdout = origOut
		os.Stderr = origErr
	}()
	os.Stdout = wout
	os.Stderr = werr

	var err error
	go func() {
		err = fn()
		wout.Close()
		werr.Close()
	}()
	stdout, _ := io.ReadAll(rout)
	stderr, _ := io.ReadAll(rerr)
	return string(stdout), string(stderr), err
}

var _ = Describe("Delete cluster", func() {
	var (
		t         *test.TestingRuntime
		clusterId string
	)

	BeforeEach(func() {
		t = test.NewTestRuntime()
		clusterId = test.MockClusterID
		args.bestEffort = false
		args.watch = false
		interactive.SetEnabled(false)
		arguments.DisableRegionDeprecationWarning = false
	})

	Context("runWithRuntime", func() {
		stubConfirm := func(string, ...interface{}) bool { return true }
		stubNoConfirm := func(string, ...interface{}) bool { return false }
		stubLogs := func(string) {}

		BeforeEach(func() {
			args.bestEffort = false
			args.watch = false
			interactive.SetEnabled(false)
			arguments.DisableRegionDeprecationWarning = false
		})

		It("runs the non-STS happy path and prints the uninstall log hint", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS()))
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

			stdout, stderr, err := captureRun(func() error {
				return runWithRuntime(t.RosaRuntime, stubConfirm, stubLogs)
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring("will start uninstalling"))
			Expect(stdout).To(ContainSubstring("rosa logs uninstall -c"))
			Expect(stdout).To(ContainSubstring("--watch"))
		})

		It("returns cleanly when deletion is not confirmed", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS()))
			})
			t.SetCluster(clusterId, clusterReady)

			stdout, stderr, err := captureRun(func() error {
				return runWithRuntime(t.RosaRuntime, stubNoConfirm, stubLogs)
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(BeEmpty())
		})

		It("prints the best-effort warning and passes the flag through", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS()))
			})
			t.SetCluster(clusterId, clusterReady)
			args.bestEffort = true

			statusBody := fmt.Sprintf(`{
				"kind": "ClusterStatus",
				"id": "%s",
				"state": "ready"
			}`, clusterId)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, statusBody))
			t.ApiServer.AppendHandlers(RespondWithJSON(
				http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{clusterReady})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))

			stdout, stderr, err := captureRun(func() error {
				return runWithRuntime(t.RosaRuntime, stubConfirm, stubLogs)
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stderr).To(ContainSubstring("best effort"))
			Expect(stderr).To(ContainSubstring("certain resources may be left behind"))
			Expect(stdout).To(ContainSubstring("will start uninstalling"))
		})

		It("prints STS cleanup guidance for clusters with operator roles", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/Installer").
						OIDCEndpointURL("https://oidc.example.com").
						OperatorRolePrefix("my-prefix").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/op-role"),
						),
				))
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

			stdout, stderr, err := captureRun(func() error {
				return runWithRuntime(t.RosaRuntime, stubConfirm, stubLogs)
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring("Operator IAM Roles:"))
			Expect(stdout).To(ContainSubstring("arn:aws:iam::123456789012:role/op-role"))
			Expect(stdout).To(ContainSubstring("OIDC Provider : https://oidc.example.com"))
			Expect(stdout).To(ContainSubstring("rosa delete operator-roles -c"))
			Expect(stdout).To(ContainSubstring("rosa delete oidc-provider -c"))
		})

		It("prints STS cleanup guidance without operator role output when none remain", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/Installer").
						OIDCEndpointURL("https://oidc-no-roles.example.com"),
				))
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

			stdout, stderr, err := captureRun(func() error {
				return runWithRuntime(t.RosaRuntime, stubConfirm, stubLogs)
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).NotTo(ContainSubstring("Operator IAM Roles:"))
			Expect(stdout).To(ContainSubstring("OIDC Provider : https://oidc-no-roles.example.com"))
			Expect(stdout).To(ContainSubstring("rosa delete operator-roles -c"))
			Expect(stdout).To(ContainSubstring("rosa delete oidc-provider -c"))
		})

		It("runs uninstall logs when watch is enabled and restores the deprecation warning flag", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS()))
			})
			t.SetCluster(clusterId, clusterReady)
			args.watch = true

			statusBody := fmt.Sprintf(`{
				"kind": "ClusterStatus",
				"id": "%s",
				"state": "ready"
			}`, clusterId)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, statusBody))
			t.ApiServer.AppendHandlers(RespondWithJSON(
				http.StatusOK, test.FormatClusterList([]*cmv1.Cluster{clusterReady})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))

			var watchedClusterKey string
			var sawDisabledWarning bool
			watchStub := func(clusterKey string) {
				watchedClusterKey = clusterKey
				sawDisabledWarning = arguments.DisableRegionDeprecationWarning
			}

			stdout, stderr, err := captureRun(func() error {
				return runWithRuntime(t.RosaRuntime, stubConfirm, watchStub)
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring("will start uninstalling"))
			Expect(watchedClusterKey).To(Equal(clusterId))
			Expect(sawDisabledWarning).To(BeTrue())
			Expect(arguments.DisableRegionDeprecationWarning).To(BeFalse())
			Expect(stdout).NotTo(ContainSubstring("rosa logs uninstall -c"))
		})

		It("returns the already-uninstalling path through the command wrapper", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS()))
			})
			t.SetCluster(clusterId, clusterReady)

			statusBody := fmt.Sprintf(`{
				"kind": "ClusterStatus",
				"id": "%s",
				"state": "uninstalling"
			}`, clusterId)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, statusBody))

			stdout, stderr, err := captureRun(func() error {
				return runWithRuntime(t.RosaRuntime, stubConfirm, stubLogs)
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring("already uninstalling"))
			Expect(stdout).To(ContainSubstring("rosa logs uninstall -c"))
		})

		It("returns an error from GetClusterState through the command wrapper", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS()))
			})
			t.SetCluster(clusterId, clusterReady)

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, ""))

			stdout, stderr, err := captureRun(func() error {
				return runWithRuntime(t.RosaRuntime, stubConfirm, stubLogs)
			})
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("failed to delete cluster"))
			Expect(err.Error()).To(ContainSubstring("expected response content type"))
		})

		It("returns a delete error through the command wrapper", func() {
			clusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS()))
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

			stdout, stderr, err := captureRun(func() error {
				return runWithRuntime(t.RosaRuntime, stubConfirm, stubLogs)
			})
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("failed to delete cluster"))
			Expect(err.Error()).To(ContainSubstring("forbidden"))
		})
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
