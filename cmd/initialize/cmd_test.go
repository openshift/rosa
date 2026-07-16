package initialize

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

const (
	supportedRegionsResponse   = `{"kind":"CloudRegionList","page":1,"size":1,"total":1,"items":[{"id":"us-east-1"}]}`
	unsupportedRegionsResponse = `{"kind":"CloudRegionList","page":1,"size":1,"total":1,"items":[{"id":"us-west-2"}]}`
	stsPoliciesResponse        = `{"kind":"AWSSTSPolicyList","page":1,"size":0,"total":0,"items":[]}`
)

var _ = Describe("initialize command", func() {
	Context("Command structure", func() {
		It("Rejects extra arguments via cobra.NoArgs", func() {
			Expect(Cmd.Args).NotTo(BeNil())
			err := Cmd.Args(Cmd, []string{"unexpected"})
			Expect(err).To(HaveOccurred())
		})

		It("Has expected flags", func() {
			Expect(Cmd.Flags().Lookup("delete")).NotTo(BeNil())
			Expect(Cmd.Flags().Lookup("disable-scp-checks")).NotTo(BeNil())
			Expect(Cmd.Flags().Lookup("use-local-credentials")).NotTo(BeNil())
			Expect(Cmd.Flags().Lookup("token")).NotTo(BeNil())
		})
	})

	Context("runWithRuntime", func() {
		var (
			ctrl           *gomock.Controller
			mockAWS        *aws.MockClient
			mockCF         *aws.MockClient
			t              *test.TestingRuntime
			permissionsRan int
			quotaRan       int
			verifyOCRan    int
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockAWS = aws.NewMockClient(ctrl)
			mockCF = aws.NewMockClient(ctrl)
			t = test.NewTestRuntime()
			Expect(os.Setenv("AWS_REGION", "us-east-1")).To(Succeed())
			DeferCleanup(os.Unsetenv, "AWS_REGION")

			args.dlt = false
			args.disableSCPChecks = false
			args.sts = false
			args.region = ""
			args.useLocalCredentials = false

			permissionsRan = 0
			quotaRan = 0
			verifyOCRan = 0
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		newDeps := func(loginErr error, confirmResult bool) initDeps {
			return initDeps{
				loginCall: func(_ *cobra.Command, _ []string, _ reporter.Logger) error {
					return loginErr
				},
				buildAWSClient: func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
					return mockAWS, nil
				},
				buildCFClient: func(_ reporter.Logger, _ *logrus.Logger, _ []string, _ bool) aws.Client {
					return mockCF
				},
				confirmPrompt: func(_ string, _ ...interface{}) bool {
					return confirmResult
				},
				runPermissions: func(_ *cobra.Command, _ []string) {
					permissionsRan++
				},
				runQuota: func(_ *cobra.Command, _ []string) {
					quotaRan++
				},
				runVerifyOC: func(_ *cobra.Command, _ []string) {
					verifyOCRan++
				},
			}
		}

		It("Returns an error when login fails", func() {
			deps := newDeps(fmt.Errorf("login failed"), true)

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("login failed"))
		})

		It("Returns an error for an unsupported region", func() {
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, unsupportedRegionsResponse),
			)
			deps := newDeps(nil, true)

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported region"))
		})

		It("Returns the supported-region retrieval error immediately", func() {
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusInternalServerError, "{}"),
			)
			deps := newDeps(nil, true)

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("retrieving supported regions"))
		})

		It("Returns an error when ValidateCredentials fails", func() {
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
			)
			deps := newDeps(nil, true)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(false, fmt.Errorf("credentials invalid"))
				return mockAWS, nil
			}

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("credentials invalid"))
		})

		It("Returns an error when ValidateCredentials returns false", func() {
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
			)
			deps := newDeps(nil, true)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(false, nil)
				return mockAWS, nil
			}

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AWS credentials are invalid"))
		})

		It("Returns exit-zero sentinel when delete confirmation is declined", func() {
			args.dlt = true
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
			)
			deps := newDeps(nil, false)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
				return mockAWS, nil
			}

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(errors.Is(err, errInitExitZero)).To(BeTrue())
		})

		It("Returns an error when delete mode fails in deleteStack", func() {
			args.dlt = true
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
				RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","page":1,"size":0,"total":0,"items":[]}`),
			)
			deps := newDeps(nil, true)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
				return mockAWS, nil
			}
			deps.buildCFClient = func(_ reporter.Logger, _ *logrus.Logger, _ []string, _ bool) aws.Client {
				mockCF.EXPECT().GetCreator().Return(&aws.Creator{ARN: "arn:aws:iam::123:user/test", AccountID: "123"}, nil)
				mockCF.EXPECT().DeleteOsdCcsAdminUser(aws.OsdCcsAdminStackName).Return(fmt.Errorf("delete failed"))
				return mockCF
			}

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("delete failed"))
		})

		It("Returns exit-zero sentinel when delete mode succeeds", func() {
			args.dlt = true
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
				RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","page":1,"size":0,"total":0,"items":[]}`),
			)
			deps := newDeps(nil, true)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
				return mockAWS, nil
			}
			deps.buildCFClient = func(_ reporter.Logger, _ *logrus.Logger, _ []string, _ bool) aws.Client {
				mockCF.EXPECT().GetCreator().Return(&aws.Creator{ARN: "arn:aws:iam::123:user/test", AccountID: "123"}, nil)
				mockCF.EXPECT().DeleteOsdCcsAdminUser(aws.OsdCcsAdminStackName).Return(nil)
				return mockCF
			}

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(errors.Is(err, errInitExitZero)).To(BeTrue())
		})

		It("Skips permissions when disable-scp-checks is set", func() {
			args.disableSCPChecks = true
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
				RespondWithJSON(http.StatusCreated, `{"kind":"Cluster","id":"dry-run-id","name":"rosa-init"}`),
			)
			deps := newDeps(nil, true)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
				return mockAWS, nil
			}
			deps.buildCFClient = func(_ reporter.Logger, _ *logrus.Logger, _ []string, _ bool) aws.Client {
				mockCF.EXPECT().EnsureOsdCcsAdminUser(aws.OsdCcsAdminStackName, aws.AdminUserName, "us-east-1").Return(true, nil)
				mockCF.EXPECT().GetCreator().Return(&aws.Creator{ARN: "arn:aws:iam::123:user/test", AccountID: "123"}, nil)
				mockCF.EXPECT().GetLocalAWSAccessKeys().Return(&aws.AccessKey{AccessKeyID: "AKIA123", SecretAccessKey: "secret"}, nil)
				return mockCF
			}

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).NotTo(HaveOccurred())
			Expect(permissionsRan).To(Equal(0))
			Expect(quotaRan).To(Equal(1))
			Expect(verifyOCRan).To(Equal(1))
		})

		It("Runs permissions, quota, and verify oc on the normal path", func() {
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
				RespondWithJSON(http.StatusOK, stsPoliciesResponse),
				RespondWithJSON(http.StatusCreated, `{"kind":"Cluster","id":"dry-run-id","name":"rosa-init"}`),
			)
			deps := newDeps(nil, true)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
				mockAWS.EXPECT().ValidateSCP(gomock.Any(), gomock.Any()).Return(true, nil)
				return mockAWS, nil
			}
			deps.buildCFClient = func(_ reporter.Logger, _ *logrus.Logger, _ []string, _ bool) aws.Client {
				mockCF.EXPECT().EnsureOsdCcsAdminUser(aws.OsdCcsAdminStackName, aws.AdminUserName, "us-east-1").Return(true, nil)
				mockCF.EXPECT().GetCreator().Return(&aws.Creator{ARN: "arn:aws:iam::123:user/test", AccountID: "123"}, nil)
				mockCF.EXPECT().GetLocalAWSAccessKeys().Return(&aws.AccessKey{AccessKeyID: "AKIA123", SecretAccessKey: "secret"}, nil)
				return mockCF
			}

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).NotTo(HaveOccurred())
			Expect(permissionsRan).To(Equal(1))
			Expect(quotaRan).To(Equal(1))
			Expect(verifyOCRan).To(Equal(1))
		})

		It("Returns an error when EnsureOsdCcsAdminUser fails", func() {
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
			)
			deps := newDeps(nil, true)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
				return mockAWS, nil
			}
			deps.buildCFClient = func(_ reporter.Logger, _ *logrus.Logger, _ []string, _ bool) aws.Client {
				mockCF.EXPECT().EnsureOsdCcsAdminUser(aws.OsdCcsAdminStackName, aws.AdminUserName, "us-east-1").Return(false, fmt.Errorf("ensure failed"))
				return mockCF
			}

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ensure failed"))
		})

		It("Returns an error when admin SCP validation fails", func() {
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
				RespondWithJSON(http.StatusOK, stsPoliciesResponse),
			)
			deps := newDeps(nil, true)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
				mockAWS.EXPECT().ValidateSCP(gomock.Any(), gomock.Any()).Return(false, fmt.Errorf("scp blocked"))
				return mockAWS, nil
			}
			deps.buildCFClient = func(_ reporter.Logger, _ *logrus.Logger, _ []string, _ bool) aws.Client {
				mockCF.EXPECT().EnsureOsdCcsAdminUser(aws.OsdCcsAdminStackName, aws.AdminUserName, "us-east-1").Return(true, nil)
				return mockCF
			}

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("scp blocked"))
		})

		It("Returns an error when ValidateSCP returns true with an error", func() {
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
				RespondWithJSON(http.StatusOK, stsPoliciesResponse),
			)
			deps := newDeps(nil, true)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
				mockAWS.EXPECT().ValidateSCP(gomock.Any(), gomock.Any()).Return(true, fmt.Errorf("scp API error"))
				return mockAWS, nil
			}
			deps.buildCFClient = func(_ reporter.Logger, _ *logrus.Logger, _ []string, _ bool) aws.Client {
				mockCF.EXPECT().EnsureOsdCcsAdminUser(aws.OsdCcsAdminStackName, aws.AdminUserName, "us-east-1").Return(true, nil)
				return mockCF
			}

			err := runWithRuntime(t.RosaRuntime, Cmd, []string{}, deps)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("scp API error"))
		})

		It("Continues with a warning when simulateCluster fails", func() {
			stdoutCapture := func(deps initDeps) (string, string, error) {
				return test.RunWithOutputCapture(func(r *rosa.Runtime, _ *cobra.Command) error {
					return runWithRuntime(r, Cmd, []string{}, deps)
				}, t.RosaRuntime, Cmd)
			}

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, supportedRegionsResponse),
				RespondWithJSON(http.StatusOK, stsPoliciesResponse),
				RespondWithJSON(http.StatusBadRequest, `{"kind":"Error","id":"400","code":"CLUSTERS-MGMT-400","reason":"dry run failed"}`),
			)
			deps := newDeps(nil, true)
			deps.buildAWSClient = func(_ *logrus.Logger, _ string, _ bool) (aws.Client, error) {
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
				mockAWS.EXPECT().ValidateSCP(gomock.Any(), gomock.Any()).Return(true, nil)
				return mockAWS, nil
			}
			deps.buildCFClient = func(_ reporter.Logger, _ *logrus.Logger, _ []string, _ bool) aws.Client {
				mockCF.EXPECT().EnsureOsdCcsAdminUser(aws.OsdCcsAdminStackName, aws.AdminUserName, "us-east-1").Return(false, nil)
				mockCF.EXPECT().GetCreator().Return(&aws.Creator{ARN: "arn:aws:iam::123:user/test", AccountID: "123"}, nil)
				mockCF.EXPECT().GetLocalAWSAccessKeys().Return(&aws.AccessKey{AccessKeyID: "AKIA123", SecretAccessKey: "secret"}, nil)
				return mockCF
			}

			stdout, stderr, err := stdoutCapture(deps)
			Expect(err).NotTo(HaveOccurred())
			Expect(stderr).To(ContainSubstring("Cluster creation failed"))
			Expect(stdout).To(ContainSubstring("Ensuring cluster administrator user"))
			Expect(verifyOCRan).To(Equal(1))
		})
	})

	Context("deleteStack", func() {
		var (
			ctrl      *gomock.Controller
			mockAWS   *aws.MockClient
			t         *test.TestingRuntime
			ocmClient *ocm.Client
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockAWS = aws.NewMockClient(ctrl)
			t = test.NewTestRuntime()
			ocmClient = t.RosaRuntime.OCMClient
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("Returns error when GetCreator fails", func() {
			mockAWS.EXPECT().GetCreator().Return(nil, fmt.Errorf("STS error"))

			err := deleteStack(mockAWS, ocmClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get AWS creator"))
		})

		It("Returns error when HasClusters fails", func() {
			mockAWS.EXPECT().GetCreator().Return(&aws.Creator{
				ARN:       "arn:aws:iam::123:user/test",
				AccountID: "123",
			}, nil)

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusInternalServerError, "{}"),
			)

			err := deleteStack(mockAWS, ocmClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to check for clusters"))
		})

		It("Returns error when user still has clusters", func() {
			mockAWS.EXPECT().GetCreator().Return(&aws.Creator{
				ARN:       "arn:aws:iam::123:user/test",
				AccountID: "123",
			}, nil)

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","page":1,"size":1,"total":1,"items":[{"id":"abc","name":"mycluster"}]}`),
			)

			err := deleteStack(mockAWS, ocmClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("user still has clusters"))
		})

		It("Returns error when DeleteOsdCcsAdminUser fails", func() {
			mockAWS.EXPECT().GetCreator().Return(&aws.Creator{
				ARN:       "arn:aws:iam::123:user/test",
				AccountID: "123",
			}, nil)

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","page":1,"size":0,"total":0,"items":[]}`),
			)

			mockAWS.EXPECT().DeleteOsdCcsAdminUser(aws.OsdCcsAdminStackName).
				Return(fmt.Errorf("stack delete failed"))

			err := deleteStack(mockAWS, ocmClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to delete user"))
		})

		It("Succeeds when no clusters exist and delete works", func() {
			mockAWS.EXPECT().GetCreator().Return(&aws.Creator{
				ARN:       "arn:aws:iam::123:user/test",
				AccountID: "123",
			}, nil)

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{"kind":"ClusterList","page":1,"size":0,"total":0,"items":[]}`),
			)

			mockAWS.EXPECT().DeleteOsdCcsAdminUser(aws.OsdCcsAdminStackName).Return(nil)

			err := deleteStack(mockAWS, ocmClient)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("simulateCluster", func() {
		var (
			ctrl      *gomock.Controller
			mockAWS   *aws.MockClient
			t         *test.TestingRuntime
			ocmClient *ocm.Client
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockAWS = aws.NewMockClient(ctrl)
			t = test.NewTestRuntime()
			ocmClient = t.RosaRuntime.OCMClient
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("Returns error when GetCreator fails", func() {
			mockAWS.EXPECT().GetCreator().Return(nil, fmt.Errorf("caller identity error"))

			err := simulateCluster(mockAWS, ocmClient, "us-east-1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get AWS creator"))
		})

		It("Returns error when GetLocalAWSAccessKeys fails", func() {
			mockAWS.EXPECT().GetCreator().Return(&aws.Creator{
				ARN:       "arn:aws:iam::123:user/test",
				AccountID: "123",
			}, nil)
			mockAWS.EXPECT().GetLocalAWSAccessKeys().Return(nil, fmt.Errorf("no keys"))

			err := simulateCluster(mockAWS, ocmClient, "us-east-1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get AWS access key"))
		})

		It("Returns error when CreateCluster dry run fails", func() {
			mockAWS.EXPECT().GetCreator().Return(&aws.Creator{
				ARN:       "arn:aws:iam::123:user/test",
				AccountID: "123",
			}, nil)
			mockAWS.EXPECT().GetLocalAWSAccessKeys().Return(&aws.AccessKey{
				AccessKeyID:     "AKIA123",
				SecretAccessKey: "secret",
			}, nil)

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusBadRequest, `{"kind":"Error","id":"400","code":"CLUSTERS-MGMT-400","reason":"dry run failed"}`),
			)

			err := simulateCluster(mockAWS, ocmClient, "us-east-1")
			Expect(err).To(HaveOccurred())
		})

		It("Succeeds with a valid dry run", func() {
			mockAWS.EXPECT().GetCreator().Return(&aws.Creator{
				ARN:       "arn:aws:iam::123:user/test",
				AccountID: "123",
			}, nil)
			mockAWS.EXPECT().GetLocalAWSAccessKeys().Return(&aws.AccessKey{
				AccessKeyID:     "AKIA123",
				SecretAccessKey: "secret",
			}, nil)

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusCreated, `{"kind":"Cluster","id":"dry-run-id","name":"rosa-init"}`),
			)

			err := simulateCluster(mockAWS, ocmClient, "us-east-1")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Defaults to us-east-1 when region is empty", func() {
			mockAWS.EXPECT().GetCreator().Return(&aws.Creator{
				ARN:       "arn:aws:iam::123:user/test",
				AccountID: "123",
			}, nil)
			mockAWS.EXPECT().GetLocalAWSAccessKeys().Return(&aws.AccessKey{
				AccessKeyID:     "AKIA123",
				SecretAccessKey: "secret",
			}, nil)

			t.ApiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/clusters_mgmt/v1/clusters"),
					func(w http.ResponseWriter, req *http.Request) {
						var body map[string]interface{}
						Expect(json.NewDecoder(req.Body).Decode(&body)).To(Succeed())
						regionBody, ok := body["region"].(map[string]interface{})
						Expect(ok).To(BeTrue())
						Expect(regionBody["id"]).To(Equal("us-east-1"))
						w.WriteHeader(http.StatusCreated)
						_, err := w.Write([]byte(`{"kind":"Cluster","id":"dry-run-id","name":"rosa-init"}`))
						Expect(err).NotTo(HaveOccurred())
					},
				),
			)

			err := simulateCluster(mockAWS, ocmClient, "")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
