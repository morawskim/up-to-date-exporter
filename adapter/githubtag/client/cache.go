package client

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/common/log"
)

type CachedGithubTagClient struct {
	githubTagsClient GithubTagClient
	cacheClient      *cache.Cache
}

func (c *CachedGithubTagClient) GetTags(repository string) ([]GithubTag, error) {
	key := fmt.Sprintf("gt:%s", repository)

	cached, found := c.cacheClient.Get(key)
	if found {
		log.Debugf("using result from cache for %s", key)

		return cached.([]GithubTag), nil //nolint: forcetypeassert
	}
	log.Debugf("using result from API for %s", key)
	live, err := c.githubTagsClient.GetTags(repository)
	c.cacheClient.Set(key, live, cache.DefaultExpiration)

	return live, err
}

func NewCachedClient(githubTagsClient GithubTagClient, cacheClient *cache.Cache) *CachedGithubTagClient {
	return &CachedGithubTagClient{
		githubTagsClient: githubTagsClient,
		cacheClient:      cacheClient,
	}
}
