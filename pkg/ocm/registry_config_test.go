package ocm

import (
	"bytes"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

var _ = Describe("Allowlist", func() {

	var ssoServer, apiServer *ghttp.Server
	var ocmClient *Client
	var body string
	var allowlist *cmv1.RegistryAllowlist

	BeforeEach(func() {
		// Create the servers:
		ssoServer = MakeTCPServer()
		apiServer = MakeTCPServer()
		apiServer.SetAllowUnhandledRequests(true)
		apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)

		// Create the token:
		accessToken := MakeTokenString("Bearer", 15*time.Minute)

		// Prepare the server:
		ssoServer.AppendHandlers(
			RespondWithAccessToken(accessToken),
		)
		// Prepare the logger:
		logger, err := logging.NewGoLoggerBuilder().
			Debug(true).
			Build()
		Expect(err).To(BeNil())
		// Set up the connection with the fake config
		connection, err := sdk.NewConnectionBuilder().
			Logger(logger).
			Tokens(accessToken).
			URL(apiServer.URL()).
			Build()
		// Initialize client object
		Expect(err).To(BeNil())
		ocmClient = &Client{ocm: connection}

		allowlist, body, err = CreateAllowlist()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Close the servers:
		ssoServer.Close()
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("KO: fails to get allowlist if returns error", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusBadRequest,
				body,
			),
		)

		_, err := ocmClient.GetAllowlist("id")
		Expect(err).To(HaveOccurred())
	})

	It("OK: gets allowlist when it exists", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				body,
			),
		)

		output, err := ocmClient.GetAllowlist("allowlist-id")

		Expect(err).To(BeNil())
		Expect(output).To(Not(BeNil()))
		Expect(output.ID()).To(Equal(allowlist.ID()))
	})
})

var _ = Describe("Registry Config", func() {
	Context("BuildAllowedRegistriesForImport", func() {
		It("OK: should pass if the user passes a valid string", func() {
			obj, err := BuildAllowedRegistriesForImport("abc.com:true")
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).To(Equal(map[string]bool{
				"abc.com": true,
			}))

		})
		It("OK: should pass if the user passes a valid long string", func() {
			obj, err := BuildAllowedRegistriesForImport("abc.com:true,efg.io:false,test.com:true")
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).To(Equal(map[string]bool{
				"abc.com":  true,
				"efg.io":   false,
				"test.com": true,
			}))
		})
		It("KO: should fail if the user does not pass the correct regex", func() {
			_, err := BuildAllowedRegistriesForImport("abc")
			Expect(err).To(MatchError("invalid identifier 'abc' for 'allowed registries for import.' " +
				"Should be in a <registry>:<boolean> format. " +
				"The boolean indicates whether the registry is secure or not."))

		})
	})

	Context("BuildRegistryConfig", func() {
		It("Returns expected output", func() {
			spec := Spec{}
			spec.AllowedRegistries = []string{"example.com", "quay.io"}
			spec.BlockedRegistries = []string{"insecure.com"}
			spec.InsecureRegistries = []string{"insecure.com"}

			additionalTrustedCa := map[string]string{}
			additionalTrustedCa["userRegistry.io"] = "-----BEGIN CERTIFICATE-----\n/abc\n-----END CERTIFICATE-----"
			spec.AdditionalTrustedCa = additionalTrustedCa
			spec.PlatformAllowlist = "abc.com"
			spec.AllowedRegistriesForImport = "quay.io:true"

			output, err := BuildRegistryConfig(spec)
			Expect(err).To(Not(HaveOccurred()))
			expectedOutput := cmv1.NewClusterRegistryConfig().
				AdditionalTrustedCa(additionalTrustedCa).
				PlatformAllowlist(cmv1.NewRegistryAllowlist().ID(spec.PlatformAllowlist)).
				AllowedRegistriesForImport(cmv1.NewRegistryLocation().
					Insecure(true).DomainName("quay.io")).
				RegistrySources(cmv1.NewRegistrySources().InsecureRegistries("insecure.com").
					BlockedRegistries("insecure.com").
					AllowedRegistries([]string{"example.com", "quay.io"}...))
			Expect(output).To(Equal(expectedOutput))
		})
	})
})

func CreateAllowlist() (*cmv1.RegistryAllowlist, string, error) {
	builder := cmv1.NewRegistryAllowlist()
	allowlist, err := builder.ID("allowlist-id").
		Registries([]string{"quay.io", "registry.redhat.io"}...).Build()
	if err != nil {
		return &cmv1.RegistryAllowlist{}, "", err
	}

	var buf bytes.Buffer
	err = cmv1.MarshalRegistryAllowlist(allowlist, &buf)

	if err != nil {
		return &cmv1.RegistryAllowlist{}, "", err
	}

	return allowlist, buf.String(), nil
}
