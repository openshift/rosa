package token

import (
	"bytes"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/cmd/config/set"
)

func TestTokenCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa token command")
}

var _ = Describe("Token", Ordered, func() {
	var buf *bytes.Buffer
	var tmpdir string
	var err error
	var ssoServer *Server

	BeforeAll(func() {
		tmpdir, err = os.MkdirTemp("/tmp", ".ocm-config-*")
		Expect(err).To(BeNil())
		os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")
	})

	AfterAll(func() {
		os.Setenv("OCM_CONFIG", "")
	})

	BeforeEach(func() {
		// Create the server
		ssoServer = MakeTCPServer()
	})

	AfterEach(func() {
		ssoServer.Close()
	})

	When("Logged in", func() {
		var accessToken string
		var refreshToken string

		BeforeEach(func() {
			buf = new(bytes.Buffer)
			writer = buf
			// Create the tokens:
			accessToken = MakeTokenString("Bearer", 10*time.Minute)
			refreshToken = MakeTokenString("Refresh", 10*time.Hour)
			ssoServer.AppendHandlers(RespondWithAccessToken(accessToken))

			err = set.SaveConfig("access_token", accessToken)
			Expect(err).To(BeNil())
			err = set.SaveConfig("refresh_token", refreshToken)
			Expect(err).To(BeNil())
			err = set.SaveConfig("token_url", ssoServer.URL())
			Expect(err).To(BeNil())
		})

		It("Displays current access token", func() {
			Cmd.Run(Cmd, []string{})
			Expect(buf.String()).To(ContainSubstring(accessToken))
		})

		It("Displays current refresh token", func() {
			args.refresh = true
			Cmd.Run(Cmd, []string{})
			Expect(buf.String()).To(ContainSubstring(refreshToken))
		})
	})
})
