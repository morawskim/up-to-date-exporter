package githubrelease

import (
	"github.com/caarlos0/version_exporter/client"
	"github.com/caarlos0/version_exporter/collector"
	"github.com/caarlos0/version_exporter/config"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	config2 "up-to-date-exporter/config"
)

type GithubReleasesCollector struct {
	prometheus.Collector
	releaseConfig *config.Config
}

func (g *GithubReleasesCollector) ReloadConfiguration(config *config2.Config) {
	g.releaseConfig.Repositories = config.GithubReleases
}

func Register(githubToken string, repositories map[string]string, cacheClient *cache.Cache) config2.ReloadCollectorConfiguration {
	releaseClient := client.NewCachedClient(
		client.NewClient(githubToken),
		cacheClient,
	)

	var releaseConfig config.Config
	releaseConfig.Repositories = repositories

	collector := collector.NewVersionCollector(&releaseConfig, releaseClient)
	prometheus.MustRegister(collector)

	return &GithubReleasesCollector{
		Collector:     collector,
		releaseConfig: &releaseConfig,
	}
}
