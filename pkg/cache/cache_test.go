package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("rosaCache", func() {
	var (
		cache    RosaCache
		testData string
	)

	BeforeEach(func() {
		spec := RosaCacheSpec{}
		cache = NewRosaCache(spec)
		testData = "testData"
		err := os.Unsetenv("RosaConfigDir")
		Expect(err).To(BeNil())
	})

	Context("Set", func() {
		It("should set the value in the cache with the specified expiration time", func() {
			key := "testKey"
			expirationTime := time.Now().Add(1 * time.Hour)

			cache.Set(key, testData, expirationTime)

			result, hasResult := cache.Get(key)
			Expect(hasResult).To(BeTrue())
			Expect(result).To(Equal(testData))
			item, ok := cache.(*rosaCache).items[key]
			Expect(ok).To(BeTrue())
			Expect(item.Object).To(Equal(testData))
			Expect(item.Expiration).To(Equal(expirationTime))
		})

		It("should set the value in the cache with the default expiration time if expiration time is zero", func() {
			key := "testKey"
			cache.Set(key, testData, time.Time{})

			result, hasResult := cache.Get(key)
			Expect(hasResult).To(BeTrue())
			Expect(result).To(Equal(testData))
			item, ok := cache.(*rosaCache).items[key]
			Expect(ok).To(BeTrue())
			Expect(item.Object).To(Equal(testData))
			Expect(item.Expiration).To(BeTemporally("~", DefaultCacheExpiration, time.Second))
		})
	})

	Context("Get", func() {
		It("should return the value from the cache if the key exists and has not expired", func() {
			key := "testKey"
			expirationTime := time.Now().Add(1 * time.Hour)

			cache.Set(key, testData, expirationTime)

			value, ok := cache.Get(key)
			Expect(ok).To(BeTrue())
			Expect(value).To(Equal(testData))
		})

		It("should return false and nil if the key does not exist in the cache", func() {
			key := "nonexistentKey"

			value, ok := cache.Get(key)
			Expect(ok).To(BeFalse())
			Expect(value).To(BeNil())
		})

		It("should return false and nil if the key has expired", func() {
			key := "testKey"
			expirationTime := time.Now().Add(-1 * time.Hour)

			cache.Set(key, testData, expirationTime)

			value, ok := cache.Get(key)
			Expect(ok).To(BeFalse())
			Expect(value).To(BeNil())
		})
	})

	Context("Items", func() {
		It("should return a map of non-expired items in the cache", func() {
			key1 := "testKey1"
			key2 := "testKey2"
			expirationTime := time.Now().Add(1 * time.Hour)

			cache.Set(key1, testData, expirationTime)
			cache.Set(key2, testData, time.Now().Add(-1*time.Hour)) // Expired item

			items := cache.Items()

			Expect(items).To(HaveKey(key1))
			Expect(items).ToNot(HaveKey(key2))
		})
	})

	Context("Dir", func() {
		When("RosaConfigDir is set", func() {
			It("should return the custom config directory path", func() {
				customConfigDir := "/tmp/config"
				err := createDirIfNotExists(customConfigDir)
				Expect(err).To(BeNil())

				err = os.Setenv(RosaConfigDir, customConfigDir)
				Expect(err).To(BeNil())

				configDir, err := cache.Dir()
				Expect(err).NotTo(HaveOccurred())
				Expect(configDir).To(Equal(fmt.Sprintf("%s/%s", customConfigDir, GobName)))

				err = deleteIfExists(customConfigDir)
				Expect(err).To(BeNil())

				err = os.Unsetenv(RosaConfigDir)
				Expect(err).To(BeNil())
			})
		})

		When("RosaConfigDir is not set", func() {
			It("should return the default config directory if file exists", func() {
				homeDir, _ := os.UserHomeDir()
				expectedPath := filepath.Join(homeDir, ".config", "ocm", GobName)

				// ensure the .config file exists
				err := createFileIfNotExists(expectedPath)
				Expect(err).To(BeNil())

				configDir, err := cache.Dir()
				Expect(err).NotTo(HaveOccurred())
				Expect(configDir).To(Equal(expectedPath))
			})

			When("the default config directory path does not exist", func() {
				It("should create a new temp cache file if ocm config does not exist", func() {
					cacheFilePath := fmt.Sprintf("/tmp/ocm/%s", GobName)

					// nuke the cache gobs it they exist
					homeDir, err := os.UserHomeDir()
					Expect(err).To(BeNil())
					err = deleteIfExists(filepath.Join(homeDir, ".config", "ocm", GobName))
					Expect(err).To(BeNil())
					err = deleteIfExists(cacheFilePath)
					Expect(err).To(BeNil())

					Expect(pathExists(cacheFilePath)).To(BeFalse())

					configDir, err := cache.Dir()
					Expect(err).NotTo(HaveOccurred())
					Expect(configDir).To(Equal(cacheFilePath))
				})
			})
		})
	})
})

func deleteIfExists(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("error deleting file: %v", err)
	}
	return nil
}

func createFileIfNotExists(filePath string) error {
	if _, err := os.Stat(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	_, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	return nil
}

func createDirIfNotExists(dirPath string) error {
	if _, err := os.Stat(dirPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("error creating path: %v", err)
	}
	return nil
}
