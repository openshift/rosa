package userrole

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("create user-role", func() {
	var (
		t      *test.TestingRuntime
		tmpdir string
	)

	saveTestConfig := func() {
		Expect(os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")).To(Succeed())
		cfg := &config.Config{
			AccessToken: MakeTokenString("Bearer", 15*time.Minute),
			URL:         "https://api.openshift.com",
			TokenURL:    t.SsoServer.URL(),
		}
		Expect(config.Save(cfg)).To(Succeed())
	}

	BeforeEach(func() {
		t = test.NewTestRuntime()
		t.RosaRuntime.Creator = &aws.Creator{
			ARN:       "arn:aws:iam::123456789012:user/test",
			AccountID: "123456789012",
			Partition: "aws",
		}

		var err error
		tmpdir, err = os.MkdirTemp("", "rosa-create-userrole-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpdir)

		args = struct {
			prefix              string
			permissionsBoundary string
			path                string
		}{}
		interactive.SetEnabled(false)
		interactive.SetModeKey("")
		Expect(Cmd.Flags().Set("mode", "")).To(Succeed())
	})

	Context("runWithRuntime", func() {
		It("returns error when mode is invalid", func() {
			interactive.SetModeKey("invalid_mode")
			Expect(Cmd.Flags().Set("mode", "invalid_mode")).To(Succeed())

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid mode"))
		})

		It("returns error when prefix is invalid", func() {
			saveTestConfig()
			interactive.SetModeKey(interactive.ModeManual)
			Expect(Cmd.Flags().Set("mode", interactive.ModeManual)).To(Succeed())
			args.prefix = "invalid prefix!"

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid role prefix matching"))
		})
	})

	Context("buildCommands", func() {
		It("generates create-role and link commands", func() {
			creator := &aws.Creator{
				AccountID: "123456789012",
				Partition: "aws",
			}

			commands := buildCommands("ManagedOpenShift", "", "testuser", creator, "production", "")
			Expect(commands).To(ContainSubstring("create-role"))
			Expect(commands).To(ContainSubstring("ManagedOpenShift-User-testuser-Role"))
			Expect(commands).To(ContainSubstring("file://sts_ocm_user_trust_policy.json"))
			Expect(commands).To(ContainSubstring("rosa link user-role --role-arn"))
		})
	})

	Context("generateUserRolePolicyFiles", func() {
		var (
			tempDir    string
			originalWd string
			policies   map[string]*cmv1.AWSSTSPolicy
		)

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "rosa-userrole-policy-test-*")
			Expect(err).NotTo(HaveOccurred())

			originalWd, err = os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			err = os.Chdir(tempDir)
			Expect(err).NotTo(HaveOccurred())

			trustPolicy, err := (&cmv1.AWSSTSPolicyBuilder{}).
				Details(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow"}]}`).
				Build()
			Expect(err).NotTo(HaveOccurred())
			policies = map[string]*cmv1.AWSSTSPolicy{
				"sts_user_ocm_trust_policy": trustPolicy,
			}
		})

		AfterEach(func() {
			err := os.Chdir(originalWd)
			Expect(err).NotTo(HaveOccurred())
			err = os.RemoveAll(tempDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("writes the trust policy file to the current directory", func() {
			reporter := reporter.CreateReporter()
			err := generateUserRolePolicyFiles(reporter, "production", "aws", "acct-123", policies)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat("sts_ocm_user_trust_policy.json")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
