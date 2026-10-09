package dnsdomains

import (
	"bytes"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
	"github.com/openshift/rosa/tests/utils/helper"
)

var _ = Describe("Create Platform API DNS domain", func() {
	BeforeEach(func() { original := args; DeferCleanup(func() { args = original }) })
	DescribeTable("reserves an HCP domain", func(explicit bool) {
		r, server := test.NewHyperfleetTestRuntime()
		cmd := &cobra.Command{}
		cmd.Flags().BoolVar(&args.hostedCp, "hosted-cp", false, "")
		if explicit {
			Expect(cmd.Flags().Set("hosted-cp", "true")).To(Succeed())
		}
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", "/api/v0/dns_domains"),
			ghttp.VerifyJSON(`{"cluster_arch":"hcp"}`),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, map[string]any{"id": "abcd.0.example.com", "cluster_arch": "hcp"}),
		))
		stdout, _, err := test.RunWithOutputCapture(func(r *rosa.Runtime, cmd *cobra.Command) error { return createDNSDomainV2(cmd, r) }, r, cmd)
		Expect(err).NotTo(HaveOccurred())
		id, err := helper.ExtractDNSDomainID(*bytes.NewBufferString(stdout))
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("abcd.0.example.com"))
		Expect(r.OCMClient).To(BeNil())
	}, Entry("default architecture", false), Entry("explicit hosted-cp", true))
	It("rejects explicitly requested Classic domains before calling the API", func() {
		r, server := test.NewHyperfleetTestRuntime()
		cmd := &cobra.Command{}
		cmd.Flags().BoolVar(&args.hostedCp, "hosted-cp", false, "")
		Expect(cmd.Flags().Set("hosted-cp", "false")).To(Succeed())
		Expect(createDNSDomainV2(cmd, r)).To(MatchError(ContainSubstring("only HCP")))
		Expect(server.ReceivedRequests()).To(BeEmpty())
	})
	It("returns API errors without reporting creation", func() {
		r, server := test.NewHyperfleetTestRuntime()
		server.AppendHandlers(ghttp.RespondWith(http.StatusServiceUnavailable, `{"message":"DNS suffix not configured"}`))
		stdout, _, err := test.RunWithOutputCapture(func(r *rosa.Runtime, cmd *cobra.Command) error { return createDNSDomainV2(cmd, r) }, r, &cobra.Command{})
		Expect(err).To(HaveOccurred())
		Expect(stdout).NotTo(ContainSubstring("has been created"))
	})
})
