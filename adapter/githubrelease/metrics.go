package githubrelease

import (
	"github.com/caarlos0/version_exporter/client"
	"github.com/caarlos0/version_exporter/collector"
	"github.com/caarlos0/version_exporter/config"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	appconfig "up-to-date-exporter/config"
)

type GithubReleasesCollector struct {
	prometheus.Collector
	releaseConfig *config.Config
}

func (g *GithubReleasesCollector) ReloadConfiguration(config *appconfig.Config) {
	g.releaseConfig.Repositories = config.GithubReleases
}

func Register( //nolint:ireturn
	githubToken string,
	repositories map[string]string,
	cacheClient *cache.Cache,
) appconfig.ReloadCollectorConfiguration {
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
