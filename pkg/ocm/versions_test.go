package ocm

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test/ci"
)

// nolint
const NonSupportedHypershiftVersionsListResponse = `{
	"kind": "VersionList",
	"href": "/api/clusters_mgmt/v1/versions?order=default+desc%2C+id+desc&page=1&search=enabled+%3D+%27true%27+AND+rosa_enabled+%3D+%27true%27+AND+channel_group+%3D+%27stable%27&size=100",
	"page": 1,
	"size": 2,
	"total": 2,
	"items": [{
		"kind": "Version",
		"href": "/api/clusters_mgmt/v1/versions/4.13.0",
		"id": "4.13.0",
		"name": "4.13.0",
		"raw_id": "4.13.0",
		"release_image": "4.14.9",
		"hosted_control_plane_default": true,
		"hosted_control_plane_enabled": false,
		"channel_group": "stable",
		"rosa_enabled": true
	}]
}`

// nolint
const VersionsListResponse = `{
	"kind": "VersionList",
	"href": "/api/clusters_mgmt/v1/versions?order=default+desc%2C+id+desc&page=1&search=enabled+%3D+%27true%27+AND+rosa_enabled+%3D+%27true%27+AND+channel_group+%3D+%27stable%27&size=100",
	"page": 1,
	"size": 2,
	"total": 2,
	"items": [{
		"kind": "Version",
		"href": "/api/clusters_mgmt/v1/versions/4.14.9",
		"id": "4.14.9",
		"name": "4.14.9",
		"raw_id": "4.14.9",
		"release_image": "4.14.9",
		"hosted_control_plane_default": true,
		"hosted_control_plane_enabled": true,
		"channel_group": "stable",
		"rosa_enabled": true
	}]
}`

var _ = Describe("Get version list", func() {
	var ssoServer, apiServer *ghttp.Server
	var ocmClient *Client

	Context("Describe version list", func() {

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
		})

		AfterEach(func() {
			// Close the servers:
			ssoServer.Close()
			apiServer.Close()
			Expect(ocmClient.Close()).To(Succeed())
		})

		It("Expects a version list", ci.Critical, func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					VersionsListResponse,
				),
			)

			vs, err := ocmClient.GetVersionsWithProduct("", DefaultChannelGroup, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(vs)).To(Equal(1))
		})

		It("Expects a valid Hypershift Version", ci.Critical, func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					VersionsListResponse,
				),
			)

			vs, err := ocmClient.ValidateHypershiftVersion("4.14.9", DefaultChannelGroup)
			Expect(err).ToNot(HaveOccurred())
			Expect(vs).To(BeTrue())
		})

		It("Expects a non supported Hypershift Version", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					NonSupportedHypershiftVersionsListResponse,
				),
			)

			vs, err := ocmClient.ValidateHypershiftVersion("4.13.0", DefaultChannelGroup)
			Expect(err).ToNot(HaveOccurred())
			Expect(vs).To(BeFalse())
		})
	})
})

var _ = Describe("Versions", Ordered, func() {

	Context("when creating a HyperShift cluster", ci.Critical, func() {
		DescribeTable("Should correctly validate the minimum version with a given channel group",
			validateVersion,
			Entry("OK: When the minimum version is provided",
				func() string { return LowestHostedCpSupport },
				func() string { return DefaultChannelGroup },
				true, true, nil),
			Entry("KO: Nightly channel group but too old",
				func() string { return "4.11.0-0.nightly-2022-10-17-040259-nightly" },
				func() string { return NightlyChannelGroup }, false, false, nil),
			Entry("OK: Nightly channel group and good version",
				func() string { return "4.14.1-0.nightly-2022-11-25-185455-nightly" },
				func() string { return NightlyChannelGroup }, true, true, nil),
			Entry("OK: When a greater version than the minimum is provided",
				func() string { return "4.15.0" },
				func() string { return DefaultChannelGroup }, true, true, nil),
			Entry("KO: When the minimum version requirement is not met",
				func() string { return "4.11.5" },
				func() string { return DefaultChannelGroup }, false, false, nil),
			Entry("OK: When a greater RC version than the minimum is provided",
				func() string { return "4.14.1-rc.1" },
				func() string { return "candidate" }, true, true, nil),
		)
	})

	Context("when listing machinepools versions", func() {
		DescribeTable("Parse correctly raw versions from version id",
			func(versionId string, expected string) {
				rawId := GetRawVersionId(versionId)
				Expect(rawId).To(Equal(expected))
			},
			Entry("stable channel",
				"openshift-v4.10.21",
				"4.10.21",
			),
			Entry("candidate channel",
				"openshift-v4.11.0-fc.0-candidate",
				"4.11.0-fc.0",
			),
			Entry("nightly channel",
				"openshift-v4.7.0-0.nightly-2021-05-21-224816-nightly",
				"4.7.0-0.nightly-2021-05-21-224816",
			),
		)
	})

	Context("when upgrading a hosted control plane", ci.Critical, func() {
		DescribeTable("Should validate the requested version with the available upgrades",
			func(userRequestedVersion string, supportedVersion string, clusterVersion string, expected bool) {
				isValid, err := IsValidVersion(userRequestedVersion, supportedVersion, clusterVersion)
				Expect(err).ToNot(HaveOccurred())
				Expect(isValid).To(Equal(expected))
			},
			Entry("From 4.14.0-rc.4 to 4.14.0", "4.14.0", "4.14.0", "4.14.0-rc.4", true),
			Entry("From 4.14.0-rc.4 to 4.14.0", "4.14.1", "4.14.0", "4.14.0-rc.4", false),
		)

		DescribeTable("Should check and parse the requested version with the available upgrades",
			func(availableUpgrades []string, version string, clusterVersion string, expected string) {
				mockCluster, err := cmv1.NewCluster().
					ID("test-id").
					Name("test-name").
					OpenshiftVersion("").
					Version(cmv1.NewVersion().RawID(clusterVersion)).Build()

				Expect(err).ToNot(HaveOccurred())
				parsedVersion, parsedErr := CheckAndParseVersion(availableUpgrades, version, mockCluster)
				Expect(parsedErr).ToNot(HaveOccurred())
				Expect(parsedVersion).To(Equal(expected))
			},
			Entry("From 4.14.0-rc.4 to 4.14.0",
				[]string{"4.14.1", "4.14.0"},
				"4.14.0",
				"4.14.0-rc.4",
				"4.14.0",
			),
			Entry("From 4.14.0-rc.4 to 4.14.1",
				[]string{"4.14.1", "4.14.0"},
				"4.14.1",
				"4.14.0-rc.4",
				"4.14.1",
			),
			Entry("From 4.14.0 to 4.14.1",
				[]string{"4.14.1"},
				"4.14.1",
				"4.14.0",
				"4.14.1",
			),
		)
	})
})

var _ = Describe("Minimal http tokens required version", ci.High, Ordered, func() {

	Context("validate http tokens required version", func() {
		DescribeTable("validate http tokens required version",
			validateMinimumHttpTokenRequiredVersion,
			Entry("required with lower version",
				"4.10",
				cmv1.Ec2MetadataHttpTokensRequired, fmt.Errorf("version '%s' is not supported with http tokens required, "+
					"minimum supported version is %s", "4.10", LowestHttpTokensRequiredSupport),
			),
			Entry("required with minimal version",
				LowestHttpTokensRequiredSupport, cmv1.Ec2MetadataHttpTokensRequired, nil,
			),
			Entry("optional with lower version",
				"4.10.21", cmv1.Ec2MetadataHttpTokensOptional, nil,
			),
			Entry("bad version",
				"bad version", cmv1.Ec2MetadataHttpTokensRequired, fmt.Errorf("version '%s' "+
					"is not supported: %v", "bad version", "Malformed version: bad version"),
			),
		)
	})
})

func validateMinimumHttpTokenRequiredVersion(version string, httpToken cmv1.Ec2MetadataHttpTokens, expectedErr error) {
	err := ValidateHttpTokensVersion(version, string(httpToken))
	if expectedErr != nil {
		Expect(err).To(BeEquivalentTo(expectedErr))
		return
	}
	Expect(err).NotTo(HaveOccurred())
}

func validateVersion(version func() string, channelGroup func() string, hypershiftEnabled bool,
	expectedValidation bool, expectedErr error) {

	v, err := cmv1.NewVersion().ID(version()).RawID(version()).HostedControlPlaneEnabled(hypershiftEnabled).Build()
	Expect(err).NotTo(HaveOccurred())

	b, err := HasHostedCPSupport(v)
	if expectedErr != nil {
		Expect(err).To(BeEquivalentTo(expectedErr))
	}
	Expect(b).To(BeIdenticalTo(expectedValidation))
}
