package dnsdomains

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"sigs.k8s.io/yaml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("List Platform API DNS domains", func() {
	BeforeEach(func() {
		original := args
		DeferCleanup(func() { args = original; output.SetOutput("") })
		output.SetOutput("")
	})
	DescribeTable("lists account reservations for the existing filters", func(all, hostedCp bool) {
		args.all, args.hostedCp = all, hostedCp
		r, server := test.NewHyperfleetTestRuntime()
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/api/v0/dns_domains", ""),
			ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]any{"items": []map[string]any{{"id": "abcd.0.example.com", "cluster_arch": "hcp", "user_defined": true, "reserved_at_timestamp": "2026-10-08T12:00:00Z"}}}),
		))
		var buffer bytes.Buffer
		Expect(listDNSDomainsV2(r, &buffer)).To(Succeed())
		domains, err := rosacli.NewClient().OCMResource.ReflectDNSDomainList(buffer)
		Expect(err).NotTo(HaveOccurred())
		domain := domains.GetDNSDomain("abcd.0.example.com")
		Expect(domain.ID).To(Equal("abcd.0.example.com"))
		Expect(domain.Architecture).To(Equal("hcp"))
		Expect(domain.UserDefined).To(Equal("Yes"))
		Expect(domain.ClusterID).To(BeEmpty())
		Expect(domain.ReservedTime).To(Equal("2026-10-08T12:00:00Z"))
		Expect(r.OCMClient).To(BeNil())
	}, Entry("default", false, false), Entry("all", true, false), Entry("hosted-cp", false, true), Entry("all hosted-cp", true, true))
	DescribeTable("preserves structured list output", func(format string, empty bool) {
		r, server := test.NewHyperfleetTestRuntime()
		body := `{"items":[{"id":"abcd.0.example.com","cluster_arch":"hcp","user_defined":true}]}`
		if empty {
			body = `{"items":[]}`
		}
		server.AppendHandlers(ghttp.RespondWith(http.StatusOK, body, http.Header{"Content-Type": {"application/json"}}))
		output.SetOutput(format)
		stdout, _, err := test.RunWithOutputCapture(func(r *rosa.Runtime, _ *cobra.Command) error { return listDNSDomainsV2(r, os.Stdout) }, r, nil)
		Expect(err).NotTo(HaveOccurred())
		data := []byte(stdout)
		if format == "yaml" {
			data, err = yaml.YAMLToJSON(data)
			Expect(err).NotTo(HaveOccurred())
		}
		var domains []hyperfleet.DNSDomain
		Expect(json.Unmarshal(data, &domains)).To(Succeed())
		if empty {
			Expect(domains).To(BeEmpty())
		} else {
			Expect(domains).To(HaveLen(1))
			Expect(domains[0].ID).To(Equal("abcd.0.example.com"))
			Expect(domains[0].UserDefined).To(BeTrue())
		}
	}, Entry("JSON", "json", false), Entry("YAML", "yaml", false), Entry("empty JSON", "json", true), Entry("empty YAML", "yaml", true))
	It("reports an empty account", func() {
		r, server := test.NewHyperfleetTestRuntime()
		server.AppendHandlers(ghttp.RespondWith(http.StatusOK, `{"items":[]}`, http.Header{"Content-Type": {"application/json"}}))
		stdout, _, err := test.RunWithOutputCapture(func(r *rosa.Runtime, _ *cobra.Command) error { return listDNSDomainsV2(r, os.Stdout) }, r, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("There are no DNS Domains for your account"))
	})
	It("returns API errors", func() {
		r, server := test.NewHyperfleetTestRuntime()
		server.AppendHandlers(ghttp.RespondWith(http.StatusForbidden, `{"message":"forbidden"}`))
		Expect(listDNSDomainsV2(r, &bytes.Buffer{})).NotTo(Succeed())
	})
	It("renders an assigned service-managed domain", func() {
		var buffer bytes.Buffer
		domain := hyperfleet.DNSDomain{ID: "abcd.0.example.com", ClusterArch: "hcp", ReservedAtTimestamp: time.Now(), Cluster: &struct {
			ID string `json:"id"`
		}{ID: "cluster-id"}}
		Expect(printDNSDomainsV2(&buffer, []hyperfleet.DNSDomain{domain})).To(Succeed())
		Expect(buffer.String()).To(ContainSubstring("cluster-id"))
		Expect(buffer.String()).To(ContainSubstring("No"))
	})
})
