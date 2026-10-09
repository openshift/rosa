package test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/openshift/rosa/pkg/hyperfleet"
)

var _ = Describe("HyperFleet test runtime", func() {
	It("makes signed account-scoped requests without OCM or local credentials", func() {
		r, server := NewHyperfleetTestRuntime()
		Expect(r.OCMClient).To(BeNil())
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/api/v0/dns_domains"),
			func(_ http.ResponseWriter, req *http.Request) {
				Expect(req.Header.Get("Authorization")).To(HavePrefix("AWS4-HMAC-SHA256 Credential=test/"))
				Expect(req.Header.Get("X-Amz-Account-Id")).To(Equal("123456789012"))
			},
			ghttp.RespondWith(http.StatusOK, `{"items":[]}`, http.Header{"Content-Type": {"application/json"}}),
		))
		domains, err := hyperfleet.NewDNSDomainClient(r.HyperFleetClient).List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(domains).To(BeEmpty())
	})
})
