package dockerimage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"up-to-date-exporter/adapter/dockerimage/client"
	"up-to-date-exporter/config"

	"github.com/Masterminds/semver"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ErrNoVersions = errors.New("no found any versions")
)

func Register(containers map[string]string, cacheClient *cache.Cache) config.ReloadCollectorConfiguration { //nolint:ireturn,lll
	dockerHubConfig := Config{Images: containers}
	dockerHubClient := client.NewCachedClient(client.NewDockerHubClient(), cacheClient)

	collector := newCollector(&dockerHubConfig, dockerHubClient)
	prometheus.MustRegister(collector)

	return collector
}

type versionCollector struct {
	mutex        sync.Mutex
	config       *Config
	client       client.DockerHubClient
	internalData dockerImageData

	up             *prometheus.Desc
	upToDate       *prometheus.Desc
	scrapeDuration *prometheus.Desc
}

type dockerImageDataItem struct {
	isUpToDate    bool
	repo          string
	version       string
	latestVersion string
}
type dockerImageData struct {
	success  bool
	duration float64
	data     []dockerImageDataItem
}

func (v *versionCollector) ReloadConfiguration(config *config.Config) {
	v.config.Images = config.DockerImages
}

func (v *versionCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- v.up
	ch <- v.upToDate
	ch <- v.scrapeDuration
}

func (v *versionCollector) Collect(ch chan<- prometheus.Metric) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for _, item := range v.internalData.data {
		ch <- prometheus.MustNewConstMetric(
			v.upToDate,
			prometheus.GaugeValue,
			boolToFloat(item.isUpToDate),
			item.repo,
			item.version,
			item.latestVersion,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		v.up,
		prometheus.GaugeValue,
		boolToFloat(v.internalData.success),
	)
	ch <- prometheus.MustNewConstMetric(
		v.scrapeDuration,
		prometheus.GaugeValue,
		v.internalData.duration,
	)
}

func (v *versionCollector) FetchData(ctx context.Context) {
	slog.Default().Info("fetch docker image version")
	v.mutex.Lock()
	defer v.mutex.Unlock()

	var success = true
	var start = time.Now()
	index := 0
	v.internalData.data = make([]dockerImageDataItem, len(v.config.Images))

	for repo, ver := range v.config.Images {
		var log = slog.Default().With("image", repo)
		sconstraint, _ := semver.NewConstraint(ver)
		latestRelease, err := getLatest(ctx, v.client, repo)

		if err != nil {
			log.Error(fmt.Sprintf("failed to collect for %s: %s", repo, err.Error()))
			success = false

			continue
		}

		if nil == latestRelease {
			continue
		}

		var isUpToDate = sconstraint.Check(latestRelease)
		log.With("constraint", ver).
			With("latest", latestRelease).
			With("up_to_date", isUpToDate).
			Debug("checked")

		v.internalData.data[index] = dockerImageDataItem{
			isUpToDate:    isUpToDate,
			repo:          repo,
			version:       ver,
			latestVersion: latestRelease.String(),
		}
		index++
	}

	v.internalData.success = success
	v.internalData.duration = time.Since(start).Seconds()
}

func newCollector(config *Config, client client.DockerHubClient) *versionCollector {
	const namespace = "docker_hub_version"
	const subsystem = ""

	return &versionCollector{
		config: config,
		client: client,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up"),
			"Exporter is being able to talk with DockerHub API",
			nil,
			nil,
		),
		upToDate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up_to_date"),
			"Whether the image latest version is in the specified semantic versioning range",
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

func getLatest(ctx context.Context, client client.DockerHubClient, repo string) (*semver.Version, error) {
	images, err := client.Releases(ctx, repo)
	if err != nil {
		return nil, err
	}

	for _, release := range images {
		version, err := semver.NewVersion(release.Tag)
		if err != nil {
			slog.Default().With("error", err).
				With("repo", repo).
				With("tag", release.Tag).
				Error(fmt.Sprintf("failed to parse tag %s", release.Tag))

			continue
		}
		if version.Prerelease() != "" {
			continue
		}

		return version, nil
	}

	return nil, ErrNoVersions
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}

	return 0.0
}
