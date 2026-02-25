package ocm

import (
	"net/http"
	"slices"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift-online/ocm-sdk-go/testing"
)

var _ = Describe("Pkg/Ocm/Machines", func() {
	Describe("OCM Client", func() {
		Describe("GetMachineTypes", func() {
			var ocmClient *Client
			var apiServer, ssoServer *ghttp.Server
			var featureFilter FeatureFilters
			BeforeEach(func() {
				// Create the servers:
				ssoServer = testing.MakeTCPServer()
				apiServer = testing.MakeTCPServer()
				apiServer.SetAllowUnhandledRequests(true)
				apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)

				// Create the token:
				accessToken := testing.MakeTokenString("Bearer", 15*time.Minute)

				// Prepare the server:
				ssoServer.AppendHandlers(
					testing.RespondWithAccessToken(accessToken),
				)
				// Prepare the logger:
				logger, err := logging.NewGoLoggerBuilder().
					Debug(false).
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
				ocmClient = NewClientWithConnection(connection)
			})
			AfterEach(func() {
				apiServer.Close()
				ssoServer.Close()
				Expect(ocmClient.Close()).Error().NotTo(HaveOccurred())
			})
			When("feature filter is empty", func() {
				BeforeEach(func() {
					featureFilter = FeatureFilters{}
				})
				When("OCM returns an error", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							ghttp.RespondWith(http.StatusBadRequest, nil),
						)
					})
					It("should also return an error", func() {
						Expect(ocmClient.GetMachineTypes(featureFilter)).Error().To(HaveOccurred())
					})
				})
				When("OCM returns an empty list of machines", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							testing.RespondWithJSON(http.StatusOK, `{
									"items": [],
									"size": 0,
									"total": 0,
									"page": 1
								}`),
						)
					})
					It("should return an empty list of machines", func() {
						machines, err := ocmClient.GetMachineTypes(featureFilter)
						Expect(err).NotTo(HaveOccurred())
						Expect(machines.Items).To(BeEmpty())
					})
				})
				When("OCM returns an short list of machines", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							testing.RespondWithJSON(http.StatusOK, `{
									"items": [{}, {}, {}],
									"size": 3,
									"total": 3,
									"page": 1
								}`),
						)
					})
					It("should return an equally short list of machines", func() {
						machines, err := ocmClient.GetMachineTypes(featureFilter)
						Expect(err).NotTo(HaveOccurred())
						Expect(machines.Items).To(HaveLen(3))
					})
				})
				When("OCM returns multiple pages of machines", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							testing.RespondWithJSON(http.StatusOK, `{
									"items": [`+strings.Join(slices.Repeat([]string{"{}"}, 100), ",")+`],
									"size": 100,
									"total": 110,
									"page": 1
								}`),
						)
						apiServer.AppendHandlers(
							testing.RespondWithJSON(http.StatusOK, `{
									"items": [`+strings.Join(slices.Repeat([]string{"{}"}, 10), ",")+`],
									"size": 10,
									"total": 110,
									"page": 2
								}`),
						)
					})
					It("should return an aggregated list of all pages of machines", func() {
						machines, err := ocmClient.GetMachineTypes(featureFilter)
						Expect(err).NotTo(HaveOccurred())
						Expect(machines.Items).To(HaveLen(110))
					})
				})
			})
			When("feature filter is nonempty", func() {
				BeforeEach(func() {
					featureFilter = FeatureFilters{FeatureFilter{featureName: "feat"}}
				})
				When("OCM returns multiple pages of machines", func() {
					BeforeEach(func() {
						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyFormKV("search", "cloud_provider.id = 'aws' AND features.feat = 'true'"),
								testing.RespondWithJSON(http.StatusOK, `{
									"items": [`+strings.Join(slices.Repeat([]string{"{}"}, 100), ",")+`],
									"size": 100,
									"total": 110,
									"page": 1
								}`),
							),
						)
						apiServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyFormKV("search", "cloud_provider.id = 'aws' AND features.feat = 'true'"),
								testing.RespondWithJSON(http.StatusOK, `{
									"items": [`+strings.Join(slices.Repeat([]string{"{}"}, 10), ",")+`],
									"size": 10,
									"total": 110,
									"page": 2
								}`),
							),
						)
					})
					It("should return an aggregated list of all pages of machines", func() {
						machines, err := ocmClient.GetMachineTypes(featureFilter)
						Expect(err).NotTo(HaveOccurred())
						Expect(machines.Items).To(HaveLen(110))
					})
				})
			})
		})
	})
})
