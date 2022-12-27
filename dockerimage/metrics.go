package dockerimage

import (
	"github.com/Masterminds/semver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"sync"
	"time"
	"up-to-date-exporter/dockerimage/client"
	"up-to-date-exporter/dockerimage/config"
)

func Register(containers map[string]string) {
	dockerHubConfig := config.Config{Images: containers}
	dockerHubClient := client.NewDockerHubClient()

	prometheus.MustRegister(collect(&dockerHubConfig, dockerHubClient))
}

type versionCollector struct {
	mutex  sync.Mutex
	config *config.Config
	client client.Client

	up             *prometheus.Desc
	upToDate       *prometheus.Desc
	scrapeDuration *prometheus.Desc
}

func (v *versionCollector) Describe(descs chan<- *prometheus.Desc) {
	descs <- v.up
	descs <- v.upToDate
	descs <- v.scrapeDuration
}

func (v *versionCollector) Collect(ch chan<- prometheus.Metric) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	var success = true
	var start = time.Now()

	for repo, ver := range v.config.Images {
		var log = log.With("image", repo)
		sconstraint, _ := semver.NewConstraint(ver)
		latestRelease, err := getLatest(v.client, repo)

		if err != nil {
			log.Errorf("failed to collect for %s: %s", repo, err.Error())
			success = false
			continue
		}

		if nil == latestRelease {
			continue
		}

		var up = sconstraint.Check(latestRelease)
		log.With("constraint", ver).
			With("latest", latestRelease).
			With("up_to_date", up).
			Debug("checked")

		ch <- prometheus.MustNewConstMetric(
			v.upToDate,
			prometheus.GaugeValue,
			boolToFloat(up),
			repo,
			ver,
			latestRelease.String(),
		)
	}

	ch <- prometheus.MustNewConstMetric(
		v.up,
		prometheus.GaugeValue,
		boolToFloat(success),
	)
	ch <- prometheus.MustNewConstMetric(
		v.scrapeDuration,
		prometheus.GaugeValue,
		time.Since(start).Seconds(),
	)
}

func collect(config *config.Config, client client.Client) prometheus.Collector {
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

func getLatest(client client.Client, repo string) (*semver.Version, error) {
	images, err := client.Releases(repo)
	if err != nil {
		return nil, err
	}

	for _, release := range images {
		version, err := semver.NewVersion(release.Tag)
		if err != nil {
			log.With("error", err).
				With("tag", release.Tag).
				Errorf("failed to parse tag %s", release.Tag)
			continue
		}
		if version.Prerelease() != "" {
			continue
		}

		return version, nil
	}

	return nil, nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
