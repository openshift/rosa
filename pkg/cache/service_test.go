package cache

import (
	"encoding/gob"
	"os"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RosaCacheService", func() {
	var (
		ctrl           *gomock.Controller
		mockCacheStore *MockRosaCache
		cacheData      RosaCacheData
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockCacheStore = NewMockRosaCache(ctrl)

		cacheData = RosaCacheData{
			Data: map[string]Item{
				"key1": {Object: "value1"},
				"key2": {Object: "value2"},
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("saveCache", func() {
		It("should save cache data to file", func() {
			mockCacheStore.EXPECT().Dir().Return("/tmp/test_cache.gob", nil)
			mockCacheStore.EXPECT().Items().Return(cacheData.Data)

			cacheService := rosaCacheService{Cache: mockCacheStore}

			err := cacheService.saveCache()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("LoadCache", func() {
		It("should load cache data from file", func() {
			mockCacheStore.EXPECT().Dir().Return("/tmp/test_cache.gob", nil)

			cacheService := rosaCacheService{Cache: mockCacheStore}

			tmpFile, err := os.CreateTemp("", "test_cache.gob")
			Expect(err).NotTo(HaveOccurred())
			defer tmpFile.Close()

			encoder := gob.NewEncoder(tmpFile)
			err = encoder.Encode(cacheData)
			Expect(err).NotTo(HaveOccurred())

			mockCacheStore.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

			_, err = cacheService.LoadCache()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Get", func() {
		It("should call Get method of RosaCache and return the result", func() {
			expectedValue := "testValue"
			mockCacheStore.EXPECT().Get("testKey").Return(expectedValue, true)

			cacheService := rosaCacheService{Cache: mockCacheStore}

			value, ok := cacheService.Get("testKey")

			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(expectedValue))
		})
	})

	Context("Set", func() {
		It("should call Set method of RosaCache and return nil", func() {
			mockCacheStore.EXPECT().Dir().Return("/tmp/test_cache.gob", nil)
			mockCacheStore.EXPECT().Items().Return(cacheData.Data)
			mockCacheStore.EXPECT().Set("testKey", []string{"testValue"}, gomock.Any())

			cacheService := rosaCacheService{Cache: mockCacheStore}
			err := cacheService.Set("testKey", []string{"testValue"})

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("NewRosaCacheService", func() {
		It("should return a new RosaCacheService without error", func() {
			rosaCacheService, err := NewRosaCacheService()
			Expect(err).To(BeNil())
			Expect(rosaCacheService).NotTo(BeNil())
		})
	})
})
