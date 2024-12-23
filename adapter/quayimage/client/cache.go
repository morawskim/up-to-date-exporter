package client

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"log/slog"
)

type QuayCachedClient struct {
	client      QuayClient
	cacheClient *cache.Cache
}

func NewCachedClient(client QuayClient, cacheClient *cache.Cache) *QuayCachedClient {
	return &QuayCachedClient{
		client:      client,
		cacheClient: cacheClient,
	}
}

func (c *QuayCachedClient) Releases(container string) ([]Release, error) {
	key := fmt.Sprintf("quay:%s", container)

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
