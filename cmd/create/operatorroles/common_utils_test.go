// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package operatorroles

import (
	"fmt"
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
		runtime.Creator = &aws.Creator{
			Partition: testPartition,
			AccountID: "123123123123",
		}

		mockClient.EXPECT().IsPolicyExists(gomock.Any()).Return(nil, nil).AnyTimes()
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

var _ = Describe("validateIngressOperatorPolicyOverride", func() {
	var ctrl *gomock.Controller
	var runtime *rosa.Runtime
	var mockClient *aws.MockClient

	const (
		testPolicyArn    = "arn:aws:iam::123456789012:policy/test-policy"
		testSharedVpcArn = "arn:aws:iam::999888777666:role/shared-vpc-role"
		testInstallerPfx = "my-prefix"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		runtime = rosa.NewRuntime()
		mockClient = aws.NewMockClient(ctrl)
		runtime.AWSClient = mockClient
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	When("the policy does not exist", func() {
		It("returns nil without further checks", func() {
			mockClient.EXPECT().IsPolicyExists(testPolicyArn).Return(nil, fmt.Errorf("NoSuchEntity"))
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("the policy exists", func() {
		BeforeEach(func() {
			mockClient.EXPECT().IsPolicyExists(testPolicyArn).Return(nil, nil)
		})

		It("returns error when GetDefaultPolicyDocument fails", func() {
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).
				Return("", fmt.Errorf("access denied"))
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access denied"))
		})

		It("returns error when policy document is invalid JSON", func() {
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).
				Return("not-json", nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).To(HaveOccurred())
		})

		It("returns nil when policy has matching shared VPC role ARN", func() {
			doc := fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "sts:AssumeRole",
					"Resource": "%s"
				}]
			}`, testSharedVpcArn)
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).Return(doc, nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error when a later statement has an unexpected shared VPC role ARN", func() {
			differentArn := "arn:aws:iam::111111111111:role/other-role"
			doc := fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::my-bucket/*"
				}, {
					"Effect": "Allow",
					"Action": "sts:AssumeRole",
					"Resource": "%s"
				}]
			}`, differentArn)
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).Return(doc, nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unexpected shared VPC role ARN"))
		})

		It("returns error when policy has different shared VPC role ARN", func() {
			differentArn := "arn:aws:iam::111111111111:role/other-role"
			doc := fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "sts:AssumeRole",
					"Resource": "%s"
				}]
			}`, differentArn)
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).Return(doc, nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unexpected shared VPC role ARN"))
		})

		It("returns nil when policy has no sts:AssumeRole statement", func() {
			doc := `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::my-bucket/*"
				}]
			}`
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).Return(doc, nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns nil when policy has Deny effect with sts:AssumeRole", func() {
			doc := fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Deny",
					"Action": "sts:AssumeRole",
					"Resource": "%s"
				}]
			}`, "arn:aws:iam::111111111111:role/other-role")
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).Return(doc, nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).ToNot(HaveOccurred())
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
		Expect(err).NotTo(HaveOccurred(), "expected the HCP version fixture to build")
		t.ApiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/versions"),
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query().Get("product")).To(Equal(ocm.HcpProduct),
						"expected hosted operator-role creation to request product=hcp")
				},
				RespondWithJSON(http.StatusOK, test.FormatVersionList([]*cmv1.Version{version})),
			),
		)

		latestVersion, err := getLatestVersion(t.RosaRuntime.OCMClient, "candidate", true)

		Expect(err).NotTo(HaveOccurred(), "expected HCP latest-version lookup to succeed")
		Expect(latestVersion).To(Equal("5.0"), "expected latest HCP policy version 5.0")
	})

	It("returns version lookup errors", func() {
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError,
			`{"kind":"Error","code":"CLUSTERS-MGMT-500","reason":"internal error"}`))

		_, err := getLatestVersion(t.RosaRuntime.OCMClient, "candidate", true)

		Expect(err).To(HaveOccurred(), "expected HCP latest-version lookup errors to be returned")
	})
})
