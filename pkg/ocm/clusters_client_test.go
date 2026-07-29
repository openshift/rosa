package ocm

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/aws"
)

func buildTestOCMClient(apiServerURL string) *Client {
	accessToken := MakeTokenString("Bearer", 15*time.Minute)
	logger, err := logging.NewGoLoggerBuilder().Debug(true).Build()
	Expect(err).NotTo(HaveOccurred())

	connection, err := sdk.NewConnectionBuilder().
		Logger(logger).
		Tokens(accessToken).
		URL(apiServerURL).
		Build()
	Expect(err).NotTo(HaveOccurred())

	return &Client{ocm: connection}
}

func buildMinimalClusterSpec() Spec {
	return Spec{
		Name:       "test-cluster",
		Region:     "us-east-1",
		AWSCreator: &aws.Creator{ARN: "arn:aws:iam::123456789012:user/test-user", AccountID: "123456789012"},
		AWSAccessKey: &aws.AccessKey{
			AccessKeyID:     "AKIA1234567890EXAMPLE",
			SecretAccessKey: "secret",
		},
	}
}

var _ = Describe("Cluster API client behavior", func() {
	var apiServer *ghttp.Server
	var ocmClient *Client

	BeforeEach(func() {
		apiServer = MakeTCPServer()
		apiServer.SetAllowUnhandledRequests(true)
		apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)
		ocmClient = buildTestOCMClient(apiServer.URL())
	})

	AfterEach(func() {
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("handles CreateCluster when DryRun is nil", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusCreated, `{"id":"cluster-1","name":"test-cluster"}`),
		)

		spec := buildMinimalClusterSpec()

		var (
			cluster *cmv1.Cluster
			err     error
		)
		Expect(func() {
			cluster, err = ocmClient.CreateCluster(spec)
		}).NotTo(Panic())
		Expect(err).NotTo(HaveOccurred())
		Expect(cluster).NotTo(BeNil())
		Expect(cluster.ID()).To(Equal("cluster-1"))
	})

	It("paginates all results when GetClusters count is zero", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, `{
				"kind":"ClusterList",
				"page":1,
				"size":1,
				"total":2,
				"items":[{"id":"cluster-1","name":"one"}]
			}`),
			RespondWithJSON(http.StatusOK, `{
				"kind":"ClusterList",
				"page":2,
				"size":1,
				"total":2,
				"items":[{"id":"cluster-2","name":"two"}]
			}`),
		)

		clusters, err := ocmClient.GetClusters(nil, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(clusters).To(HaveLen(2))
		Expect(clusters[0].ID()).To(Equal("cluster-1"))
		Expect(clusters[1].ID()).To(Equal("cluster-2"))
	})

	It("sends dryRun query and payload when CreateCluster dry run is requested", func() {
		dryRun := true
		spec := buildMinimalClusterSpec()
		spec.DryRun = &dryRun
		spec.Version = "4.15.1"

		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query().Get("dryRun")).To(Equal("true"))
					body, err := io.ReadAll(request.Body)
					Expect(err).NotTo(HaveOccurred())

					payload := map[string]interface{}{}
					Expect(json.Unmarshal(body, &payload)).To(Succeed())
					Expect(payload["name"]).To(Equal("test-cluster"))
					Expect(payload).To(HaveKey("version"))
				},
				RespondWithJSON(http.StatusCreated, `{"id":"cluster-1","name":"test-cluster"}`),
			),
		)

		cluster, err := ocmClient.CreateCluster(spec)
		Expect(err).NotTo(HaveOccurred())
		Expect(cluster).To(BeNil())
	})

	It("updates a cluster after resolving it by key", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, `{
				"kind":"ClusterList",
				"page":1,
				"size":1,
				"total":1,
				"items":[{"id":"cluster-1","name":"one"}]
			}`),
			RespondWithJSON(http.StatusOK, `{"id":"cluster-1"}`),
		)

		err := ocmClient.UpdateCluster("cluster-1", nil, Spec{
			Version: "4.15.2",
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns empty state when GetClusterState has no body", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, `{}`),
		)

		state, err := ocmClient.GetClusterState("cluster-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(cmv1.ClusterState("")))
	})

	It("deletes a cluster after resolving it by key", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, `{
				"kind":"ClusterList",
				"page":1,
				"size":1,
				"total":1,
				"items":[{"id":"cluster-1","name":"one"}]
			}`),
			RespondWithJSON(http.StatusOK, `{"kind":"DeleteResponse"}`),
		)

		cluster, err := ocmClient.DeleteCluster("cluster-1", false, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(cluster).NotTo(BeNil())
		Expect(cluster.ID()).To(Equal("cluster-1"))
	})

	It("hibernates a cluster when org capability is enabled", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, `{
				"id":"acct-1",
				"organization":{"id":"org-1","external_id":"ext-1"}
			}`),
			RespondWithJSON(http.StatusOK, `{
				"id":"org-1",
				"capabilities":[{"name":"capability.organization.hibernate_cluster","value":"true"}]
			}`),
			RespondWithJSON(http.StatusOK, `{}`),
		)

		err := ocmClient.HibernateCluster("cluster-1")
		Expect(err).NotTo(HaveOccurred())
	})

	It("resumes a cluster when org capability is enabled", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, `{
				"id":"acct-1",
				"organization":{"id":"org-1","external_id":"ext-1"}
			}`),
			RespondWithJSON(http.StatusOK, `{
				"id":"org-1",
				"capabilities":[{"name":"capability.organization.hibernate_cluster","value":"true"}]
			}`),
			RespondWithJSON(http.StatusOK, `{}`),
		)

		err := ocmClient.ResumeCluster("cluster-1")
		Expect(err).NotTo(HaveOccurred())
	})

	It("detects clusters using a matching oidc endpoint url", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, `{
				"kind":"ClusterList",
				"page":1,
				"size":1,
				"total":1,
				"items":[{"id":"cluster-1"}]
			}`),
		)

		exists, err := ocmClient.HasAClusterUsingOidcEndpointUrl("https://issuer.example.com")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
	})
})
