package githubrelease

import (
	"github.com/caarlos0/version_exporter/client"
	"github.com/caarlos0/version_exporter/collector"
	"github.com/caarlos0/version_exporter/config"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

func Register(githubToken string, repositories map[string]string) {
	releaseClient := client.NewCachedClient(
		client.NewClient(githubToken),
		cache.New(time.Minute*15, time.Minute*15),
	)

	var releaseConfig config.Config
	releaseConfig.Repositories = repositories

	prometheus.MustRegister(collector.NewVersionCollector(&releaseConfig, releaseClient))
}
