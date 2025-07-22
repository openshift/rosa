package version

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/cache"
	"github.com/openshift/rosa/pkg/clients"
	"github.com/openshift/rosa/pkg/logging"
)

var expectedVersions = []string{"1.0.0", "2.0.1", "3.0.0"}

var htmlContent = `
			<html>
				<body>
					<div class="file"><a href="1.0.0/">1.0.0/</a></div>
					<div class="file"><a href="2.0.1/">2.0.0/</a></div>
					<div class="file"><a href="3.0.0/">3.0.0/</a></div>
				</body>
			</html>`

var _ = Describe("RetrieveLatestVersionFromMirror", func() {
	var (
		r          retriever
		mockClient *clients.MockHTTPClient
		mockCache  *cache.MockRosaCacheService
		mockCtrl   *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = clients.NewMockHTTPClient(mockCtrl)
		mockCache = cache.NewMockRosaCacheService(mockCtrl)
		logger := logging.NewLogger()

		r = retriever{
			client: mockClient,
			logger: logger,
			cache:  mockCache,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("possible versions are available from the cache", func() {
		It("should retrieve the latest version", func() {
			mockCache.EXPECT().Get(cache.VersionCacheKey).Return(expectedVersions, true)

			latestVersion, err := r.RetrieveLatestVersionFromMirror()

			Expect(err).ToNot(HaveOccurred())
			Expect(latestVersion).ToNot(BeNil())
			Expect(latestVersion.String()).To(Equal("3.0.0"))
		})
	})

	When("there are no versions available from the cache", func() {
		It("should return the latest from the mirror", func() {
			mockCache.EXPECT().Get(cache.VersionCacheKey).Return([]string{}, false).AnyTimes()
			mockCache.EXPECT().Set(cache.VersionCacheKey, expectedVersions).Return(nil).AnyTimes()

			response := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(htmlContent)),
			}
			mockClient.EXPECT().Get(gomock.Any()).Return(response, nil)

			// Call the method under test
			latestVersion, err := r.RetrieveLatestVersionFromMirror()

			// Verify the result
			Expect(err).To(BeNil())
			Expect(latestVersion).ToNot(BeNil())
			Expect(latestVersion.String()).To(Equal("3.0.0"))

		})
	})

	When("there is an error retrieving possible versions from the mirror", func() {
		It("should return an error", func() {
			mockCache.EXPECT().Get(cache.VersionCacheKey).Return([]string{}, false).AnyTimes()
			mockClient.EXPECT().Get(gomock.Any()).Return(nil, fmt.Errorf("mock error"))

			latestVersion, err := r.RetrieveLatestVersionFromMirror()

			Expect(err).To(HaveOccurred())
			Expect(latestVersion).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("problem retrieving possible versions from mirror"))
		})
	})
})

var _ = Describe("RetrievePossibleVersionsFromCache", func() {
	var (
		mockCtrl  *gomock.Controller
		mockCache *cache.MockRosaCacheService
		r         Retriever
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCache = cache.NewMockRosaCacheService(mockCtrl)
		r = NewRetriever(RetrieverSpec{
			Client: &clients.DefaultHTTPClient{},
			Logger: logrus.New(),
			Cache:  mockCache,
		})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("cache has versions", func() {
		It("should return versions and true", func() {
			mockCache.EXPECT().Get(cache.VersionCacheKey).Return(expectedVersions, true)
			versions, ok := r.RetrievePossibleVersionsFromCache()
			Expect(ok).To(BeTrue())
			Expect(versions).To(Equal(expectedVersions))
		})
	})

	When("cache does not have versions", func() {
		It("should return empty versions and false", func() {
			mockCache.EXPECT().Get(cache.VersionCacheKey).Return([]string{}, false)
			versions, ok := r.RetrievePossibleVersionsFromCache()
			Expect(ok).To(BeFalse())
			Expect(versions).To(BeEmpty())
		})
	})

	When("cache fails to convert versions", func() {
		It("should return empty versions and false", func() {
			mockCache.EXPECT().Get(cache.VersionCacheKey).Return("invalid data", true)
			versions, ok := r.RetrievePossibleVersionsFromCache()
			Expect(ok).To(BeFalse())
			Expect(versions).To(BeEmpty())
		})
	})
})

var _ = Describe("RetrievePossibleVersionsFromMirror", func() {
	var (
		mockCtrl       *gomock.Controller
		mockCache      *cache.MockRosaCacheService
		mockHttpClient *clients.MockHTTPClient
		r              Retriever
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCache = cache.NewMockRosaCacheService(mockCtrl)
		mockHttpClient = clients.NewMockHTTPClient(mockCtrl)
		logger := logging.NewLogger()

		// Instead of creating a new instance of DefaultHTTPClient directly,
		// pass the mockHttpClient to the retriever.
		r = NewRetriever(RetrieverSpec{
			Client: mockHttpClient,
			Logger: logger,
			Cache:  mockCache,
		})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("possible versions are available in cache", func() {
		It("should return versions from cache", func() {
			mockCache.EXPECT().Get(cache.VersionCacheKey).Return(expectedVersions, true)
			versions, err := r.RetrievePossibleVersionsFromMirror()
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(Equal(expectedVersions))
		})
	})

	When("possible versions are not available in cache", func() {
		It("should return versions from mirror", func() {
			mockCache.EXPECT().Get(cache.VersionCacheKey).Return([]string{}, false).AnyTimes()
			mockCache.EXPECT().Set(cache.VersionCacheKey, expectedVersions).Return(nil).AnyTimes()

			response := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(htmlContent)),
			}
			mockHttpClient.EXPECT().Get(gomock.Any()).Return(response, nil).AnyTimes()
			versions, err := r.RetrievePossibleVersionsFromMirror()
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(Equal(expectedVersions))
		})
	})
})
