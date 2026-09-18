package operatorroles

import (
	"net/http"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Create dns domain", func() {
	var ctrl *gomock.Controller
	var runtime *rosa.Runtime

	var testPartition = "test"
	var testArn = "arn:aws:iam::123456789012:role/test"
	var testVersion = "2012-10-17"
	var mockClient *aws.MockClient

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		runtime = rosa.NewRuntime()
		mockClient = aws.NewMockClient(ctrl)
		runtime.AWSClient = mockClient
		mockClient.EXPECT().GetCreator().Return(&aws.Creator{Partition: testPartition}, nil)

		mockClient.EXPECT().IsPolicyExists(gomock.Any()).Return(nil, nil).AnyTimes()

		creator, err := runtime.AWSClient.GetCreator()
		Expect(err).ToNot(HaveOccurred())
		runtime.Creator = creator
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Common Utils for create/operatorroles Test", func() {
		When("getHcpSharedVpcPolicy", func() {
			It("OK: Gets policy arn back", func() {
				returnedArn := "arn:aws:iam::123123123123:policy/test"
				mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any()).Return(returnedArn, nil)
				arn, err := getHcpSharedVpcPolicy(runtime, testArn, testVersion)
				Expect(err).ToNot(HaveOccurred())
				Expect(arn).To(Equal(returnedArn))
			})
			It("KO: Returns empty policy when fails", func() {
				mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any()).Return("", errors.UserErrorf("Failed"))
				arn, err := getHcpSharedVpcPolicy(runtime, testArn, testVersion)
				Expect(err).To(HaveOccurred())
				Expect(arn).To(Equal(""))
			})
		})
	})
})

var _ = Describe("getLatestVersion", func() {
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

		latestVersion, err := getLatestVersion(t.RosaRuntime.OCMClient, "candidate", true)

		Expect(err).NotTo(HaveOccurred())
		Expect(latestVersion).To(Equal("5.0"))
	})

	It("returns version lookup errors", func() {
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError,
			`{"kind":"Error","code":"CLUSTERS-MGMT-500","reason":"internal error"}`))

		_, err := getLatestVersion(t.RosaRuntime.OCMClient, "candidate", true)

		Expect(err).To(HaveOccurred())
	})
})
