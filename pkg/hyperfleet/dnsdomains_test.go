package hyperfleet

import (
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	hfclient "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	hfrest "github.com/openshift-online/rosa-hyperfleet-api/clientset/rest"
)

var _ = Describe("Platform API DNS domains", func() {
	var client *DNSDomainClient
	var server *ghttp.Server
	BeforeEach(func() {
		server = ghttp.NewServer()
		DeferCleanup(server.Close)
		sdkClient, err := hfclient.NewForConfig(&hfrest.Config{
			Host: server.URL() + "/stage", Region: "us-east-1", AccountID: "123456789012",
			AWSConfig: aws.Config{Credentials: credentials.NewStaticCredentialsProvider("test", "test", "")},
		})
		Expect(err).NotTo(HaveOccurred())
		client = NewDNSDomainClient(sdkClient)
	})

	It("creates an HCP reservation using the signed account-scoped transport", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", "/stage/api/v0/dns_domains"),
			ghttp.VerifyJSON(`{"cluster_arch":"hcp"}`),
			func(_ http.ResponseWriter, req *http.Request) {
				Expect(req.Header.Get("Authorization")).To(HavePrefix("AWS4-HMAC-SHA256 "))
				Expect(req.Header.Get("X-Amz-Account-Id")).To(Equal("123456789012"))
			},
			ghttp.RespondWithJSONEncoded(http.StatusCreated, map[string]any{
				"kind": "DNSDomain", "id": "abcd.0.example.com", "cluster_arch": "hcp",
				"user_defined": true, "reserved_at_timestamp": "2026-10-08T12:00:00Z",
			}),
		))
		domain, err := client.Create(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(domain.ID).To(Equal("abcd.0.example.com"))
		Expect(domain.ClusterArch).To(Equal("hcp"))
		Expect(domain.UserDefined).To(BeTrue())
		Expect(domain.ReservedAtTimestamp.IsZero()).To(BeFalse())
	})

	It("lists reservations without an OCM organization filter", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/stage/api/v0/dns_domains", ""),
			ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]any{
				"kind": "DNSDomainList", "items": []map[string]any{{"id": "abcd.0.example.com", "cluster_arch": "hcp"}},
			}),
		))
		domains, err := client.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(domains).To(HaveLen(1))
		Expect(domains[0].ID).To(Equal("abcd.0.example.com"))
	})

	It("returns an empty collection for no reservations", func() {
		server.AppendHandlers(ghttp.RespondWith(http.StatusOK, `{"items":[]}`,
			http.Header{"Content-Type": {"application/json"}}))
		domains, err := client.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(domains).To(Equal([]DNSDomain{}))
	})

	It("deletes the reservation by its domain name", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", "/stage/api/v0/dns_domains/abcd.0.example.com"),
			ghttp.RespondWith(http.StatusNoContent, nil),
		))
		Expect(client.Delete(context.Background(), "abcd.0.example.com")).To(Succeed())
	})

	It("rejects a missing ID without sending a request", func() {
		Expect(client.Delete(context.Background(), "")).To(MatchError("DNS domain ID is required"))
		Expect(server.ReceivedRequests()).To(BeEmpty())
	})

	DescribeTable("propagates failed API operations", func(operation string, status int) {
		server.AppendHandlers(ghttp.RespondWithJSONEncoded(status, map[string]any{
			"kind": "Status", "apiVersion": "v1", "status": "Failure", "message": "DNS domain request rejected",
			"reason": "Forbidden", "code": status,
		}))
		var err error
		switch operation {
		case "create":
			_, err = client.Create(context.Background())
		case "list":
			_, err = client.List(context.Background())
		case "delete":
			err = client.Delete(context.Background(), "abcd.0.example.com")
		}
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("DNS domain request rejected"))
	},
		Entry("create forbidden", "create", http.StatusForbidden),
		Entry("list failure", "list", http.StatusInternalServerError),
		Entry("delete in-use domain", "delete", http.StatusConflict),
		Entry("delete missing domain", "delete", http.StatusNotFound),
	)

	DescribeTable("rejects malformed success responses", func(operation, body string) {
		server.AppendHandlers(ghttp.RespondWith(http.StatusOK, body,
			http.Header{"Content-Type": {"application/json"}}))
		var err error
		if operation == "create" {
			_, err = client.Create(context.Background())
		} else {
			_, err = client.List(context.Background())
		}
		Expect(err).To(HaveOccurred())
	},
		Entry("invalid create JSON", "create", "invalid"),
		Entry("create missing ID", "create", `{}`),
		Entry("invalid list JSON", "list", "invalid"),
	)
})
