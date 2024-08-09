package ocm

import (
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/config"
)

var _ = Describe("OCM Client", Ordered, func() {
	When("Keeping tokens alive", Ordered, func() {
		var ssoServer, apiServer *ghttp.Server
		var ocmClient *Client
		var tmpdir string
		var err error
		accessToken := MakeTokenString("Bearer", 15*time.Minute)
		refreshToken := MakeTokenString("Refresh", 15*time.Minute)
		newAccessToken := MakeTokenString("Bearer", 15*time.Minute)
		newRefreshToken := MakeTokenString("Refresh", 15*time.Minute)

		BeforeAll(func() {
			tmpdir, err = os.MkdirTemp("/tmp", ".ocm-config-*")
			Expect(err).To(BeNil())
			os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")
		})

		AfterAll(func() {
			os.Setenv("OCM_CONFIG", "")
		})

		BeforeEach(func() {
			// Create the servers:
			ssoServer = MakeTCPServer()
			apiServer = MakeTCPServer()
			apiServer.SetAllowUnhandledRequests(true)
			apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)

			// Prepare the server:
			ssoServer.AppendHandlers(
				RespondWithAccessAndRefreshTokens(newAccessToken, newRefreshToken),
			)
			// Prepare the logger:
			logger, err := logging.NewGoLoggerBuilder().
				Debug(false).
				Build()
			Expect(err).NotTo(HaveOccurred())
			// Set up the connection with the fake config
			connection, err := sdk.NewConnectionBuilder().
				Logger(logger).
				Tokens(accessToken, refreshToken).
				URL(apiServer.URL()).
				TokenURL(ssoServer.URL()).
				Build()
			Expect(err).NotTo(HaveOccurred())

			ocmClient = NewClientWithConnection(connection)
			config.Save(&config.Config{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
			})
		})

		AfterEach(func() {
			ssoServer.Close()
			apiServer.Close()
		})

		It("Fails with inability to get tokens", func() {
			config.Save(&config.Config{})
			connection, _ := sdk.NewConnectionBuilder().
				Tokens(refreshToken).
				URL(apiServer.URL()).
				TokenURL(ssoServer.URL()).
				Build()
			ssoServer.Reset()
			ssoServer.AllowUnhandledRequests = true
			ssoServer.UnhandledRequestStatusCode = http.StatusInternalServerError
			ocmClient = NewClientWithConnection(connection)
			err = ocmClient.KeepTokensAlive()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Can't get new tokens"))
		})

		It("Fails without a valid connection", func() {
			ocmClient = NewClientWithConnection(nil)
			err = ocmClient.KeepTokensAlive()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Connection is nil"))
		})

		It("Persists updated tokens", func() {
			myconf, err := config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(myconf).NotTo(BeNil())
			Expect(myconf.AccessToken).To(Equal(accessToken))
			Expect(myconf.RefreshToken).To(Equal(refreshToken))

			err = ocmClient.KeepTokensAlive()
			Expect(err).NotTo(HaveOccurred())

			myconf, err = config.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(myconf.AccessToken).To(Equal(newAccessToken))
			Expect(myconf.RefreshToken).To(Equal(newRefreshToken))
		})
	})
})
