package whoami

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("whoami command", func() {
	var t *test.TestingRuntime
	var tmpdir string

	BeforeEach(func() {
		t = test.NewTestRuntime()
		Expect(os.Setenv("AWS_REGION", "us-east-1")).To(Succeed())
		DeferCleanup(os.Unsetenv, "AWS_REGION")

		var err error
		tmpdir, err = os.MkdirTemp("", ".ocm-whoami-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpdir)

		output.SetOutput("")
		DeferCleanup(output.SetOutput, "")
	})

	saveConfig := func(cfg *config.Config) {
		Expect(os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")).To(Succeed())
		DeferCleanup(os.Unsetenv, "OCM_CONFIG")
		Expect(config.Save(cfg)).To(Succeed())
	}

	validConfig := func() *config.Config {
		accessToken := MakeTokenString("Bearer", 15*time.Minute)
		return &config.Config{
			AccessToken: accessToken,
			ClientID:    "test-client",
			URL:         t.ApiServer.URL(),
			TokenURL:    t.SsoServer.URL(),
		}
	}

	accountResponse := func(extID string) string {
		resp := map[string]interface{}{
			"kind":       "Account",
			"id":         "acct-123",
			"first_name": "Test",
			"last_name":  "User",
			"username":   "testuser",
			"email":      "test@example.com",
			"organization": map[string]interface{}{
				"id":          "org-456",
				"name":        "Test Org",
				"external_id": extID,
			},
		}
		data, _ := json.Marshal(resp)
		return string(data)
	}

	It("Displays account information from OCM in text mode", func() {
		cfg := validConfig()
		saveConfig(cfg)

		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, accountResponse("ext-789")),
		)

		stdout, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(ContainSubstring("AWS Account ID:"))
		Expect(stdout).To(ContainSubstring("OCM Account Username:"))
		Expect(stdout).To(ContainSubstring("testuser"))
	})

	It("Displays account information in JSON mode", func() {
		cfg := validConfig()
		saveConfig(cfg)
		output.SetOutput(output.JSON)

		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, accountResponse("ext-789")),
		)

		stdout, _, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("OCM Account Username"))
	})

	It("Falls back to token data when GetCurrentAccount returns nil", func() {
		claims := MakeClaims()
		claims["first_name"] = "Token"
		claims["last_name"] = "Fallback"
		claims["username"] = "tokenuser"
		claims["email"] = "token@example.com"
		claims["org_id"] = "org-from-token"
		tokenObj := MakeTokenObject(claims)

		cfg := &config.Config{
			AccessToken: tokenObj.Raw,
			ClientID:    "test-client",
			URL:         t.ApiServer.URL(),
			TokenURL:    t.SsoServer.URL(),
		}
		saveConfig(cfg)

		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusNotFound, "{}"),
		)

		stdout, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(ContainSubstring("tokenuser"))
	})

	It("Includes org external ID when present", func() {
		cfg := validConfig()
		saveConfig(cfg)

		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, accountResponse("ext-789")),
		)

		stdout, _, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("OCM Organization External ID:"))
		Expect(stdout).To(ContainSubstring("ext-789"))
	})

	It("Omits org external ID when empty", func() {
		cfg := validConfig()
		saveConfig(cfg)

		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, accountResponse("")),
		)

		stdout, _, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).NotTo(ContainSubstring("OCM Organization External ID"))
	})

	It("Returns error when config file cannot be loaded", func() {
		Expect(os.Setenv("OCM_CONFIG", tmpdir)).To(Succeed())
		DeferCleanup(os.Unsetenv, "OCM_CONFIG")

		_, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("Failed to load config file"))
	})

	It("Returns error when config is nil", func() {
		Expect(os.Setenv("OCM_CONFIG", tmpdir+"/empty_config.json")).To(Succeed())
		DeferCleanup(os.Unsetenv, "OCM_CONFIG")

		_, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("not logged in"))
	})

	It("Returns error when config is invalid", func() {
		invalidCfg := &config.Config{
			AccessToken: "",
			ClientID:    "",
			URL:         "",
			TokenURL:    "",
		}
		saveConfig(invalidCfg)

		_, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("not logged in"))
	})

	It("Returns error when Armed() fails", func() {
		cfg := &config.Config{
			AccessToken: "invalid-jwt-token",
			ClientID:    "test-client",
			URL:         t.ApiServer.URL(),
			TokenURL:    t.SsoServer.URL(),
		}
		saveConfig(cfg)

		_, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("Failed to verify configuration"))
	})

	It("Returns error when GetCurrentAccount API fails", func() {
		cfg := validConfig()
		saveConfig(cfg)

		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusInternalServerError, `{"kind":"Error","id":"500","href":"/api/accounts_mgmt/v1/errors/500","code":"ACCOUNTS-MGMT-500","reason":"internal error"}`),
		)

		_, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("Failed to get current account"))
	})

	It("Returns error when token fallback fails due to bad token data", func() {
		cfg := &config.Config{
			AccessToken: MakeTokenString("Bearer", 15*time.Minute),
			ClientID:    "test-client",
			URL:         t.ApiServer.URL(),
			TokenURL:    t.SsoServer.URL(),
		}
		saveConfig(cfg)

		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusNotFound, "{}"),
		)

		_, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("Failed to get account data from token"))
	})

	It("Returns error when structured output format is invalid", func() {
		cfg := validConfig()
		saveConfig(cfg)
		output.SetOutput("bogus-format")

		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, accountResponse("")),
		)

		_, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("unknown format"))
	})

	It("Sets AWS fields from the pre-set Creator", func() {
		cfg := validConfig()
		saveConfig(cfg)

		t.RosaRuntime.Creator = &aws.Creator{
			ARN:       "arn:aws:iam::123456789:user/testuser",
			AccountID: "123456789",
			IsSTS:     false,
		}

		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, accountResponse("")),
		)

		stdout, _, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout).To(ContainSubstring("123456789"))
		Expect(stdout).To(ContainSubstring("arn:aws:iam::123456789:user/testuser"))
	})

	It("Builds an OCM client when the runtime does not already have one", func() {
		cfg := validConfig()
		saveConfig(cfg)
		t.RosaRuntime.OCMClient = nil

		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, accountResponse("")),
		)

		stdout, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(ContainSubstring("OCM Account Username:"))
	})

	It("Rejects extra arguments via cobra.NoArgs", func() {
		Expect(Cmd.Args).NotTo(BeNil())
		err := Cmd.Args(Cmd, []string{"unexpected"})
		Expect(err).To(HaveOccurred())
	})
})
