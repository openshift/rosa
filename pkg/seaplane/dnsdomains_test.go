// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	seaplaneclient "github.com/openshift-online/rosa-seaplane-api/clientset"
	seaplanerest "github.com/openshift-online/rosa-seaplane-api/clientset/rest"
)

var _ = Describe("Platform API DNS domains", func() {
	var client *DNSDomainClient
	var server *ghttp.Server
	BeforeEach(func() {
		server = ghttp.NewServer()
		DeferCleanup(server.Close)
		sdkClient, err := seaplaneclient.NewForConfig(&seaplanerest.Config{
			Host: server.URL() + "/stage", Region: "us-east-1", AccountID: "123456789012",
			AWSConfig: aws.Config{Credentials: credentials.NewStaticCredentialsProvider("test", "test", "")},
		})
		Expect(err).NotTo(HaveOccurred())
		client = NewDNSDomainClient(sdkClient)
	})

	reservation := func(name, baseDomain string) map[string]any {
		return map[string]any{
			"kind":     "DNSReservation",
			"metadata": map[string]any{"name": name, "creationTimestamp": "2026-10-08T12:00:00Z"},
			"status":   map[string]any{"phase": "Ready", "baseDomain": baseDomain},
		}
	}

	It("creates a named reservation on the DNS reservation route", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", "/stage/api/v0/dns_reservations"),
			func(_ http.ResponseWriter, req *http.Request) {
				Expect(req.Header.Get("Authorization")).To(HavePrefix("AWS4-HMAC-SHA256 "))
				Expect(req.Header.Get("X-Amz-Account-Id")).To(Equal("123456789012"))
				var body struct {
					Metadata struct {
						Name string `json:"name"`
					} `json:"metadata"`
				}
				Expect(json.NewDecoder(req.Body).Decode(&body)).To(Succeed())
				Expect(body.Metadata.Name).To(HavePrefix("rosa-"))
			},
			ghttp.RespondWithJSONEncoded(http.StatusCreated, reservation("rosa-1a2b3c4d", "abcd.0.example.com")),
		))
		domain, err := client.Create(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(domain.ID).To(Equal("rosa-1a2b3c4d"))
		Expect(domain.BaseDomain).To(Equal("abcd.0.example.com"))
		Expect(domain.ClusterArch).To(Equal("hcp"))
		Expect(domain.UserDefined).To(BeTrue())
		Expect(domain.ReservedAtTimestamp.IsZero()).To(BeFalse())
	})

	It("maps the reservation list to OCM-shaped domains", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/stage/api/v0/dns_reservations", ""),
			ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]any{
				"kind": "DNSReservationList",
				"items": []map[string]any{
					reservation("rosa-1a2b3c4d", "abcd.0.example.com"),
					reservation("rosa-5e6f7a8b", ""),
				},
			}),
		))
		domains, err := client.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(domains).To(HaveLen(2))
		Expect(domains[0].ID).To(Equal("rosa-1a2b3c4d"))
		Expect(domains[0].BaseDomain).To(Equal("abcd.0.example.com"))
		Expect(domains[1].BaseDomain).To(BeEmpty())
	})

	It("returns an empty collection for no reservations", func() {
		server.AppendHandlers(ghttp.RespondWith(http.StatusOK, `{"items":[]}`,
			http.Header{"Content-Type": {"application/json"}}))
		domains, err := client.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(domains).To(Equal([]DNSDomain{}))
	})

	It("deletes the reservation by name", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", "/stage/api/v0/dns_reservations/rosa-1a2b3c4d"),
			ghttp.RespondWith(http.StatusNoContent, nil),
		))
		Expect(client.Delete(context.Background(), "rosa-1a2b3c4d")).To(Succeed())
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
			err = client.Delete(context.Background(), "rosa-1a2b3c4d")
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
		Entry("create missing name", "create", `{"metadata":{}}`),
		Entry("invalid list JSON", "list", "invalid"),
	)
})
