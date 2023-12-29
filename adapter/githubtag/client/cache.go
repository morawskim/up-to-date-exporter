package client

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"log/slog"
)

type CachedGithubTagClient struct {
	githubTagsClient GithubTagClient
	cacheClient      *cache.Cache
}

func (c *CachedGithubTagClient) GetTags(repository string) ([]GithubTag, error) {
	key := fmt.Sprintf("gt:%s", repository)

	cached, found := c.cacheClient.Get(key)
	if found {
		slog.Default().Debug(fmt.Sprintf("using result from cache for %s", key))

		return cached.([]GithubTag), nil //nolint: forcetypeassert
	}
	slog.Default().Debug(fmt.Sprintf("using result from API for %s", key))
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
