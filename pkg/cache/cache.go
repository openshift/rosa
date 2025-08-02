package cache

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/openshift/rosa/pkg/constants"
)

var DefaultCacheExpiration = time.Now().Add(30 * time.Minute)

const (
	GobName = "ocm-cache.gob"
)

type RosaCache interface {
	Set(k string, x interface{}, d time.Time)
	Get(k string) (interface{}, bool)
	Items() map[string]Item
	Dir() (string, error)
}

var _ RosaCache = &rosaCache{}

type RosaCacheSpec struct {
	DefaultExpiration time.Time
}

type rosaCache struct {
	defaultExpiration time.Time
	items             map[string]Item
	mu                sync.RWMutex
}

func NewRosaCache(spec RosaCacheSpec) RosaCache {
	cache := &rosaCache{
		items: make(map[string]Item),
	}
	if spec.DefaultExpiration.IsZero() {
		cache.defaultExpiration = DefaultCacheExpiration
		return cache
	}
	cache.defaultExpiration = spec.DefaultExpiration
	return cache
}

func (c *rosaCache) Set(k string, x interface{}, d time.Time) {
	if d.IsZero() {
		d = c.defaultExpiration
	}
	c.mu.Lock()
	c.items[k] = Item{
		Object:     x,
		Expiration: d,
	}
	c.mu.Unlock()
}

func (c *rosaCache) Get(k string) (interface{}, bool) {
	c.mu.RLock()
	item, found := c.items[k]
	if !found {
		c.mu.RUnlock()
		return nil, false
	}
	if !item.Expiration.IsZero() {
		if item.Expired() {
			c.mu.RUnlock()
			return nil, false
		}
	}
	c.mu.RUnlock()
	return item.Object, true
}

func (c *rosaCache) Items() map[string]Item {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := make(map[string]Item, len(c.items))
	for k, v := range c.items {
		if !v.Expiration.IsZero() {
			if v.Expired() {
				continue
			}
		}
		m[k] = v
	}
	return m
}

func (c *rosaCache) Dir() (string, error) {
	configDir, hasEnvVar, err := getConfigDirectoryEnvVar()
	if err != nil {
		return "", fmt.Errorf("error getting config directory: %v", err)
	}
	if hasEnvVar {
		return configDir, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting current user home dir: %v", err)
	}

	ocmConfigPath := filepath.Join(homeDir, ".config", "ocm", GobName)
	ocmConfigPathExists, err := pathExists(ocmConfigPath)
	if err != nil {
		return "", fmt.Errorf("error checking if path exists: %v", err)
	}
	if ocmConfigPathExists {
		return ocmConfigPath, nil
	}

	tmpDir, err := getAndEnsureOCMDirectoryExists()
	if err != nil {
		return "", fmt.Errorf("error ensuring tmp OCM directory exists: %v", err)
	}
	return fmt.Sprintf("%s/%s", tmpDir, GobName), nil
}

func getAndEnsureOCMDirectoryExists() (string, error) {
	curUser, err := user.Current()
	if err != nil {
		return "", err
	}
	ocmDir := fmt.Sprintf("%s/%s/ocm", os.TempDir(), curUser.Username)

	_, err = os.Stat(ocmDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(ocmDir, 0755); err != nil {
				return "", err
			}
			return ocmDir, nil
		}
		return "", err
	}
	return ocmDir, nil
}

func getConfigDirectoryEnvVar() (string, bool, error) {
	configDir := os.Getenv(constants.OcmConfig)
	if configDir == "" {
		return "", false, nil
	}
	fileInfo, err := os.Stat(configDir)
	if err != nil {
		// if the user passes in a path to a file and not a dir silently ignore and default to /tmp or .config
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if fileInfo.IsDir() {
		return fmt.Sprintf("%s/%s", strings.TrimRight(configDir, "/"), GobName), true, nil
	}
	return "", false, err
}
