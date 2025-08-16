package cache

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
)

const (
	VersionCacheKey = "Versions"
)

type RosaCacheData struct {
	Data map[string]Item
}

type RosaCacheService interface {
	LoadCache() (RosaCache, error)
	Get(key string) (interface{}, bool)
	Set(key string, value []string) error
}

var _ RosaCacheService = &rosaCacheService{}

type rosaCacheService struct {
	Cache RosaCache
}

func NewRosaCacheService() (RosaCacheService, error) {
	cache := &rosaCacheService{
		Cache: NewRosaCache(RosaCacheSpec{}),
	}
	_, err := cache.LoadCache()
	if err != nil {
		return cache, fmt.Errorf("failed to load cache: %v", err)
	}
	return cache, nil
}

func (r rosaCacheService) LoadCache() (RosaCache, error) {
	filePath, err := r.Cache.Dir()
	if err != nil {
		return r.Cache, err
	}
	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return r.Cache, fmt.Errorf("error opening cache file: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return r.Cache, fmt.Errorf("error getting file info: %v", err)
	}
	if stat.Size() == 0 {
		return r.Cache, nil
	}

	decoder := gob.NewDecoder(file)
	for {
		var cacheData RosaCacheData
		if err := decoder.Decode(&cacheData); err != nil {
			if err == io.EOF {
				break
			}
			return r.Cache, fmt.Errorf("error decoding cache file: %v", err)
		}
		for key, item := range cacheData.Data {
			r.Cache.Set(key, item.Object, item.Expiration)
		}
	}
	return r.Cache, nil
}

func (r rosaCacheService) Get(key string) (interface{}, bool) {
	return r.Cache.Get(key)
}

func (r rosaCacheService) Set(key string, value []string) error {
	r.Cache.Set(key, value, DefaultCacheExpiration)
	return r.saveCache()
}

func (r rosaCacheService) saveCache() error {
	filePath, err := r.Cache.Dir()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error opening cache file for writing: %v", err)
	}
	defer file.Close()

	cacheData := RosaCacheData{Data: make(map[string]Item)}
	for key, item := range r.Cache.Items() {
		cacheData.Data[key] = item
	}

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(cacheData); err != nil {
		return fmt.Errorf("error encoding cache: %v", err)
	}
	return nil
}
