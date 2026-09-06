package githubrelease

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"up-to-date-exporter/adapter/githubrelease/client"
	appconfig "up-to-date-exporter/config"

	"github.com/Masterminds/semver"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

type githubReleasesCollector struct {
	mutex         sync.Mutex
	releaseConfig *releaseConfig
	client        client.GithubReleaseClient
	internalData  githubReleaseData

	up             *prometheus.Desc
	upToDate       *prometheus.Desc
	failed         *prometheus.Desc
	scrapeDuration *prometheus.Desc
}

type releaseConfig struct {
	Repositories map[string]string
}

type githubReleaseDataItem struct {
	isUpToDate    bool
	repo          string
	version       string
	latestVersion string
}

type githubReleaseData struct {
	success  bool
	duration float64
	data     []githubReleaseDataItem
}

func (g *githubReleasesCollector) ReloadConfiguration(config *appconfig.Config) {
	g.releaseConfig.Repositories = config.GithubReleases
}

func (g *githubReleasesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- g.up
	ch <- g.upToDate
	ch <- g.failed
	ch <- g.scrapeDuration
}

func (g *githubReleasesCollector) Collect(ch chan<- prometheus.Metric) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for _, item := range g.internalData.data {
		if item.version == "" {
			ch <- prometheus.MustNewConstMetric(
				g.failed,
				prometheus.GaugeValue,
				1,
				item.repo,
			)
		} else {
			ch <- prometheus.MustNewConstMetric(
				g.failed,
				prometheus.GaugeValue,
				0,
				item.repo,
			)
			ch <- prometheus.MustNewConstMetric(
				g.upToDate,
				prometheus.GaugeValue,
				boolToFloat(item.isUpToDate),
				item.repo,
				item.version,
				item.latestVersion,
			)
		}
	}

	ch <- prometheus.MustNewConstMetric(
		g.up,
		prometheus.GaugeValue,
		boolToFloat(g.internalData.success),
	)
	ch <- prometheus.MustNewConstMetric(
		g.scrapeDuration,
		prometheus.GaugeValue,
		g.internalData.duration,
	)
}

func (g *githubReleasesCollector) FetchData(ctx context.Context) {
	slog.Default().Info("fetch github releases data")

	g.mutex.Lock()
	defer g.mutex.Unlock()

	start := time.Now()
	success := true
	index := 0
	g.internalData.data = make([]githubReleaseDataItem, len(g.releaseConfig.Repositories))

	for repo, version := range g.releaseConfig.Repositories {
		g.internalData.data[index] = githubReleaseDataItem{
			isUpToDate:    false,
			repo:          repo,
			version:       "",
			latestVersion: "",
		}
		currentIndex := index
		index++

		log := slog.Default().With("repo", repo)
		constraint, err := semver.NewConstraint(version)
		if err != nil {
			log.Error(fmt.Sprintf("failed to collect for %s: %s", repo, err.Error()))
			success = false

			continue
		}

		latestVersion, err := getLatestRelease(ctx, g.client, repo)
		if err != nil {
			log.Error(fmt.Sprintf("failed to collect for %s: %s", repo, err.Error()))
			success = false

			continue
		}

		if latestVersion == nil {
			continue
		}

		isUpToDate := constraint.Check(latestVersion)
		log.With("constraint", version).
			With("latest", latestVersion).
			With("up_to_date", isUpToDate).
			Debug("checked")

		g.internalData.data[currentIndex] = githubReleaseDataItem{
			isUpToDate:    isUpToDate,
			repo:          repo,
			version:       version,
			latestVersion: latestVersion.String(),
		}
	}

	g.internalData.success = success
	g.internalData.duration = time.Since(start).Seconds()
}

func Register( //nolint:ireturn
	githubToken string,
	repositories map[string]string,
	cacheClient *cache.Cache,
) appconfig.ReloadCollectorConfiguration {
	releaseClient := client.NewCachedClient(
		client.NewGithubReleaseHTTPClient(githubToken),
		cacheClient,
	)
	releaseConfig := &releaseConfig{Repositories: repositories}
	collector := newCollector(releaseConfig, releaseClient)
	prometheus.MustRegister(collector)

	return collector
}

func newCollector(config *releaseConfig, client client.GithubReleaseClient) *githubReleasesCollector {
	const namespace = "version"
	const subsystem = ""

	return &githubReleasesCollector{
		releaseConfig: config,
		client:        client,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up"),
			"Exporter is being able to talk with GitHub API",
			nil,
			nil,
		),
		upToDate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up_to_date"),
			"Whether the repository latest version is in the specified semantic versioning range",
			[]string{"repository", "constraint", "latest"},
			nil,
		),
		failed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "failed"),
			"Whether the repository latest tag cannot be found",
			[]string{"repository"},
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

func getLatestRelease(ctx context.Context, client client.GithubReleaseClient, repo string) (*semver.Version, error) {
	releases, err := client.Releases(ctx, repo)
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}

		version, err := semver.NewVersion(release.Tag)
		if err != nil {
			slog.Default().With("error", err).
				With("tag", release.Tag).
				Error(fmt.Sprintf("failed to parse tag %s", release.Tag))

			continue
		}

		if version.Prerelease() != "" {
			continue
		}

		return version, nil
	}

	return nil, errors.New("no found any versions")
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}

	return 0.0
}
