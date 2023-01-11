package client

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/common/log"
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
		log.Debugf("using result from cache for %s", key)

		return cached.([]Release), nil //nolint: forcetypeassert
	}
	log.Debugf("using result from API for %s", key)
	live, err := c.client.Releases(container)
	c.cacheClient.Set(key, live, cache.DefaultExpiration)

	return live, err
}
