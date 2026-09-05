package quayimage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"up-to-date-exporter/adapter/quayimage/client"
	"up-to-date-exporter/config"

	"github.com/Masterminds/semver"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ErrNoVersions = errors.New("no found any versions")
)

func Register(containers map[string]string, cacheClient *cache.Cache) config.ReloadCollectorConfiguration { //nolint:ireturn,lll
	quayConfig := Config{Images: containers}
	quayClient := client.NewCachedClient(client.NewQuayClient(), cacheClient)

	collector := newCollector(&quayConfig, quayClient)
	prometheus.MustRegister(collector)

	return collector
}

type versionCollector struct {
	mutex        sync.Mutex
	config       *Config
	client       client.QuayClient
	internalData quayData

	up             *prometheus.Desc
	upToDate       *prometheus.Desc
	scrapeDuration *prometheus.Desc
}

type quayDataItem struct {
	isUpToDate    bool
	repo          string
	version       string
	latestVersion string
}
type quayData struct {
	success  bool
	duration float64
	data     []quayDataItem
}

func (v *versionCollector) ReloadConfiguration(config *config.Config) {
	v.config.Images = config.QuaryImages
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
		if item.repo == "" {
			continue
		}
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
	v.mutex.Lock()
	defer v.mutex.Unlock()
	slog.Default().Info("fetch quay data")
	var success = true
	var start = time.Now()
	var index = 0
	v.internalData.data = make([]quayDataItem, len(v.config.Images))

	for repo, ver := range v.config.Images {
		var log = slog.Default().With("image", repo)
		sconstraint, _ := semver.NewConstraint(strings.TrimPrefix(ver, extractPrefixFromTag(ver)))
		latestRelease, err := getLatest(ctx, v.client, repo, ver)

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

		v.internalData.data[index] = quayDataItem{
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

func newCollector(config *Config, client client.QuayClient) *versionCollector {
	const namespace = "quay_version"
	const subsystem = ""

	return &versionCollector{
		config: config,
		client: client,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up"),
			"Exporter is being able to talk with quay API",
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

func getLatest(ctx context.Context, client client.QuayClient, repo string, ver string) (*semver.Version, error) {
	images, err := client.Releases(ctx, repo)
	if err != nil {
		return nil, err
	}
	prefix := extractPrefixFromTag(ver)

	for _, release := range images {
		version, err := semver.NewVersion(strings.TrimPrefix(release.Tag, prefix))
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

func extractPrefixFromTag(input string) string {
	index := strings.Index(input, "-")
	if index != -1 {
		result := input[:index+1]

		return result
	}

	return ""
}
