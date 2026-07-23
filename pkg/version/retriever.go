package version

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"

	goVer "github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/cache"
	"github.com/openshift/rosa/pkg/clients"
)

type Retriever interface {
	RetrieveLatestVersionFromMirror() (*goVer.Version, error)
	RetrievePossibleVersionsFromCache() ([]string, bool)
	RetrievePossibleVersionsFromMirror() ([]string, error)
}

func NewRetriever(spec RetrieverSpec) Retriever {
	return &retriever{
		logger: spec.Logger,
		client: spec.Client,
		cache:  spec.Cache,
	}
}

type RetrieverSpec struct {
	Logger *logrus.Logger
	Client clients.HTTPClient
	Cache  cache.RosaCacheService
}

var _ Retriever = &retriever{}

type retriever struct {
	logger *logrus.Logger
	client clients.HTTPClient
	cache  cache.RosaCacheService
}

func (r retriever) RetrieveLatestVersionFromMirror() (*goVer.Version, error) {
	possibleVersions, err := r.RetrievePossibleVersionsFromMirror()
	if err != nil {
		return nil, fmt.Errorf("there was a problem retrieving possible versions from mirror: %v", err)
	}
	possibleVersions = parseVersionURIsToVersionStreams(possibleVersions)
	if len(possibleVersions) == 0 {
		return nil, fmt.Errorf("no versions available in mirror %s", baseReleasesFolder)
	}
	latestVersion, err := goVer.NewVersion(possibleVersions[0])
	if err != nil {
		return nil, fmt.Errorf("there was a problem retrieving latest version: %v", err)
	}
	for _, ver := range possibleVersions[1:] {
		curVersion, err := goVer.NewVersion(ver)
		if err != nil {
			continue
		}
		if curVersion.GreaterThan(latestVersion) {
			latestVersion = curVersion
		}
	}
	return latestVersion, nil
}

func (r retriever) RetrievePossibleVersionsFromCache() ([]string, bool) {
	cachedVersions, hasCachedVersions := r.cache.Get(cache.VersionCacheKey)
	if !hasCachedVersions {
		return []string{}, false
	}

	possibleVersions, hasExtracted, _ := cache.ConvertToStringSlice(cachedVersions)
	if !hasExtracted {
		return []string{}, false
	}
	return possibleVersions, true
}

func (r retriever) RetrievePossibleVersionsFromMirror() ([]string, error) {
	var possibleVersions []string

	possibleVersions, gotPossibleVersionsFromCache := r.RetrievePossibleVersionsFromCache()
	if gotPossibleVersionsFromCache {
		return possibleVersions, nil
	}

	resp, err := r.client.Get(baseReleasesFolder)
	if err != nil {
		return []string{}, fmt.Errorf("error setting up request for latest released rosa cli: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return []string{},
			fmt.Errorf("error while requesting latest released rosa cli: %d %s", resp.StatusCode, resp.Status)
	}
	possibleVersions, err = parseVersionLinksFromHTML(resp.Body)
	if err != nil {
		return []string{}, fmt.Errorf("error parsing response body: %w", err)
	}
	if err := r.cache.Set(cache.VersionCacheKey, possibleVersions); err != nil {
		r.logger.Debugf("Failed to set possible versions in cache : %v", err)
	}
	r.logger.Debugf("Versions available for download: %v", possibleVersions)
	return possibleVersions, nil
}

// parseVersionLinksFromHTML extracts version hrefs from the mirror's HTML
// directory listing. It looks for <a> tags inside <tr class="file"> rows.
func parseVersionLinksFromHTML(r io.Reader) ([]string, error) {
	var versions []string
	tokenizer := html.NewTokenizer(r)
	inFileRow := false
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			if tokenizer.Err() == io.EOF {
				break
			}
			return nil, tokenizer.Err()
		}
		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			tn, hasAttr := tokenizer.TagName()
			tag := string(tn)
			if tag == "tr" && hasAttr {
				inFileRow = hasClass(tokenizer, "file")
			}
			if tag == "a" && inFileRow && hasAttr {
				if href := getAttr(tokenizer, "href"); href != "" {
					ver := strings.TrimSpace(href)
					ver = strings.TrimRight(ver, "/")
					if ver != "latest" {
						versions = append(versions, ver)
					}
				}
			}
		}
		if tt == html.EndTagToken {
			tn, _ := tokenizer.TagName()
			if string(tn) == "tr" {
				inFileRow = false
			}
		}
	}
	return versions, nil
}

func hasClass(z *html.Tokenizer, class string) bool {
	for {
		key, val, more := z.TagAttr()
		if string(key) == "class" {
			for _, tok := range strings.Fields(string(val)) {
				if tok == class {
					return true
				}
			}
		}
		if !more {
			return false
		}
	}
}

func getAttr(z *html.Tokenizer, name string) string {
	for {
		key, val, more := z.TagAttr()
		if string(key) == name {
			return string(val)
		}
		if !more {
			return ""
		}
	}
}

func parseVersionURIsToVersionStreams(uriList []string) []string {
	parsedList := make([]string, len(uriList))
	for i, uri := range uriList {
		if strings.HasPrefix(uri, "https://") {
			// Needs to be parsed, find last segment
			split := strings.Split(uri, "/")
			slashCount := strings.Count(uri, "/")

			parsedList[i] = split[slashCount]
			for len(split[slashCount]) == 0 {
				parsedList[i] = split[slashCount-1]
				slashCount--
			}
		} else {
			parsedList[i] = uri
		}
	}
	return parsedList
}
