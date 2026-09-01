package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/patrickmn/go-cache"
)

type GithubReleaseCachedClient struct {
	client      GithubReleaseClient
	cacheClient *cache.Cache
}

func NewCachedClient(githubReleaseClient GithubReleaseClient, cacheClient *cache.Cache) *GithubReleaseCachedClient {
	return &GithubReleaseCachedClient{
		client:      githubReleaseClient,
		cacheClient: cacheClient,
	}
}

func (c *GithubReleaseCachedClient) Releases(ctx context.Context, repository string) ([]GithubRelease, error) {
	key := fmt.Sprintf("gr:%s", repository)

	cached, found := c.cacheClient.Get(key)
	if found {
		slog.Default().Debug(fmt.Sprintf("using result from cache for %s", key))

		return cached.([]GithubRelease), nil //nolint: forcetypeassert
	}

	slog.Default().Debug(fmt.Sprintf("using result from API for %s", key))
	live, err := c.client.Releases(ctx, repository)
	c.cacheClient.Set(key, live, cache.DefaultExpiration)

	return live, err
}
