package client

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"log/slog"
)

type DockerHubCachedClient struct {
	client      DockerHubClient
	cacheClient *cache.Cache
}

func NewCachedClient(client DockerHubClient, cacheClient *cache.Cache) *DockerHubCachedClient {
	return &DockerHubCachedClient{
		client:      client,
		cacheClient: cacheClient,
	}
}

func (c *DockerHubCachedClient) Releases(container string) ([]Release, error) {
	key := fmt.Sprintf("dh:%s", container)

	cached, found := c.cacheClient.Get(key)
	if found {
		slog.Default().Debug(fmt.Sprintf("using result from cache for %s", key))

		return cached.([]Release), nil //nolint: forcetypeassert
	}
	slog.Default().Debug(fmt.Sprintf("using result from API for %s", key))
	live, err := c.client.Releases(container)
	c.cacheClient.Set(key, live, cache.DefaultExpiration)

	return live, err
}
