package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/patrickmn/go-cache"
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

func (c *QuayCachedClient) Releases(ctx context.Context, container string) ([]Release, error) {
	key := fmt.Sprintf("quay:%s", container)

	cached, found := c.cacheClient.Get(key)
	if found {
		slog.Default().Debug(fmt.Sprintf("using result from cache for %s", key))

		return cached.([]Release), nil //nolint: forcetypeassert
	}
	slog.Default().Debug(fmt.Sprintf("using result from API for %s", key))
	live, err := c.client.Releases(ctx, container)
	c.cacheClient.Set(key, live, cache.DefaultExpiration)

	return live, err
}
