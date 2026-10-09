package dnsdomains

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Delete Platform API DNS domain", func() {
	It("deletes the reservation and preserves the success message", func() {
		r, server := test.NewHyperfleetTestRuntime()
		server.AppendHandlers(ghttp.CombineHandlers(ghttp.VerifyRequest("DELETE", "/api/v0/dns_domains/abcd.0.example.com"), ghttp.RespondWith(http.StatusNoContent, nil)))
		stdout, _, err := test.RunWithOutputCapture(func(r *rosa.Runtime, _ *cobra.Command) error { return deleteDNSDomainV2(r, "abcd.0.example.com") }, r, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("Successfully deleted dns domain 'abcd.0.example.com'"))
		Expect(r.OCMClient).To(BeNil())
	})
	DescribeTable("propagates deletion errors without reporting success", func(status int) {
		r, server := test.NewHyperfleetTestRuntime()
		server.AppendHandlers(ghttp.RespondWith(status, `{"kind":"Status","apiVersion":"v1","status":"Failure","message":"DNS domain cannot be deleted","reason":"Conflict","code":409}`, http.Header{"Content-Type": {"application/json"}}))
		stdout, _, err := test.RunWithOutputCapture(func(r *rosa.Runtime, _ *cobra.Command) error { return deleteDNSDomainV2(r, "abcd.0.example.com") }, r, nil)
		Expect(err).To(HaveOccurred())
		Expect(stdout).NotTo(ContainSubstring("Successfully deleted"))
	}, Entry("in use", http.StatusConflict), Entry("not found", http.StatusNotFound))
})
