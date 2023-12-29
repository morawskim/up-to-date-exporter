package githubtag

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"sync"
	"time"
	"up-to-date-exporter/adapter/githubtag/client"
	"up-to-date-exporter/config"
)

type githubTagsCollector struct {
	mutex  sync.Mutex
	config *Config
	client client.GithubTagClient

	up             *prometheus.Desc
	upToDate       *prometheus.Desc
	scrapeDuration *prometheus.Desc
}

func (g *githubTagsCollector) ReloadConfiguration(config *config.Config) {
	g.config.Repositories = config.GithubTags
}

func (g *githubTagsCollector) Describe(descs chan<- *prometheus.Desc) {
	descs <- g.up
	descs <- g.upToDate
	descs <- g.scrapeDuration
}

func (g *githubTagsCollector) Collect(ch chan<- prometheus.Metric) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	start := time.Now()
	success := true

	for repo, version := range g.config.Repositories {
		var log = slog.Default().With("repo", repo)
		constraint, _ := semver.NewConstraint(version)

		latestVersion, err := getLatestTag(g.client, repo)
		if err != nil {
			log.Error(fmt.Sprintf("failed to collect for %s: %s", repo, err.Error()))
			success = false

			continue
		}

		if nil == latestVersion {
			continue
		}

		var isUpToDate = constraint.Check(latestVersion)
		log.With("constraint", version).
			With("latest", latestVersion).
			With("up_to_date", isUpToDate).
			Debug("checked")

		ch <- prometheus.MustNewConstMetric(
			g.upToDate,
			prometheus.GaugeValue,
			boolToFloat(isUpToDate),
			repo,
			version,
			latestVersion.String(),
		)
	}

	ch <- prometheus.MustNewConstMetric(
		g.up,
		prometheus.GaugeValue,
		boolToFloat(success),
	)
	ch <- prometheus.MustNewConstMetric(
		g.scrapeDuration,
		prometheus.GaugeValue,
		time.Since(start).Seconds(),
	)
}

func Register(repositories map[string]string, cacheClient *cache.Cache) config.ReloadCollectorConfiguration { //nolint:ireturn, lll
	githubTagsConfig := &Config{Repositories: repositories}
	githubTagsClient := client.NewCachedClient(
		client.NewGithubTagHTTPClient(""),
		cacheClient,
	)

	col := collector(githubTagsConfig, githubTagsClient)
	prometheus.MustRegister(col)

	return col
}

func collector(config *Config, client client.GithubTagClient) *githubTagsCollector {
	const namespace = "github_tag"
	const subsystem = ""

	return &githubTagsCollector{
		config: config,
		client: client,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up"),
			"Exporter is being able to talk with GitHub API",
			nil,
			nil,
		),
		upToDate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up_to_date"),
			"Whether the repository latest tag is in the specified semantic versioning range",
			[]string{"repository", "constraint", "latest"},
			nil,
		),
		scrapeDuration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "scrape_duration_seconds"),
			"Returns how long the probe took to complete in seconds",
			nil,
			nil,
		),
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}

	return 0.0
}

func getLatestTag(client client.GithubTagClient, repo string) (*semver.Version, error) {
	tags, err := client.GetTags(repo)

	if err != nil {
		return nil, err
	}

	for _, tag := range tags {
		version, err := semver.NewVersion(tag.Tag)
		if err != nil {
			slog.Default().With("error", err).
				With("tag", tag.Tag).
				Error(fmt.Sprintf("failed to parse tag %s", tag.Tag))

			continue
		}

		if version.Prerelease() != "" {
			continue
		}

		return version, nil
	}

	return nil, errors.New("no found any versions")
}
