package accountroles

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
)

const cmdTestExternalID = "223B9588-36A5-ECA4-BE8D-7C673B77CEC1"

var _ = Describe("validateAccountRolesSTSExternalID", func() {
	It("accepts a valid external-id", func() {
		err := validateAccountRolesSTSExternalID(cmdTestExternalID)
		Expect(err).NotTo(HaveOccurred(), "valid external-id should pass validation")
	})

	It("accepts an empty external-id", func() {
		err := validateAccountRolesSTSExternalID("")
		Expect(err).NotTo(HaveOccurred(), "empty external-id should pass validation")
	})

	It("rejects an invalid external-id", func() {
		err := validateAccountRolesSTSExternalID("x")
		Expect(err).To(HaveOccurred(), "invalid external-id should fail validation")
	})
})

var _ = Describe("getPolicyVersion", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("requests HCP versions for hosted control plane roles", func() {
		version, err := cmv1.NewVersion().
			ID("openshift-v5.0.0-candidate").
			RawID("5.0.0").
			Enabled(true).
			ROSAEnabled(true).
			ChannelGroup("candidate").
			Build()
		Expect(err).NotTo(HaveOccurred())
		t.ApiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/versions"),
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query().Get("product")).To(Equal(ocm.HcpProduct))
				},
				RespondWithJSON(http.StatusOK, test.FormatVersionList([]*cmv1.Version{version})),
			),
		)

		policyVersion, err := getPolicyVersion(t.RosaRuntime.OCMClient, "5.0", "candidate", true)

		Expect(err).NotTo(HaveOccurred())
		Expect(policyVersion).To(Equal("5.0"))
	})

	It("returns version lookup errors", func() {
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError,
			`{"kind":"Error","code":"CLUSTERS-MGMT-500","reason":"internal error"}`))

		_, err := getPolicyVersion(t.RosaRuntime.OCMClient, "5.0", "candidate", true)

		Expect(err).To(HaveOccurred())
	})
})
