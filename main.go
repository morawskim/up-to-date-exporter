package main

import (
	"fmt"
	"github.com/alecthomas/kingpin"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
	"os"
	"time"
	"up-to-date-exporter/adapter/dockerimage"
	"up-to-date-exporter/adapter/githubrelease"
	"up-to-date-exporter/adapter/githubtag"
	"up-to-date-exporter/adapter/quayimage"
	"up-to-date-exporter/config"
)

var (
	//nolint: gochecknoglobals
	bind = kingpin.Flag("bind", "addr to bind the server").Default(":9333").String()
	//nolint: gochecknoglobals
	debug = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	//nolint: gochecknoglobals
	configFile = kingpin.Flag("config.file", "config file").Default("config.yaml").ExistingFile()
	version    = "dev"
)

func main() {
	kingpin.Version("up-to-date-exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	logger.Info("starting up-to-date-exporter")

	if *debug {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)

		logger.Debug("enabled debug mode")
	}

	cacheClient := cache.New(time.Minute*15, time.Minute*15)

	var conf = config.Config{}
	var collectorGitHubReleases, collectorDockerImages, collectorGitHubTags config.ReloadCollectorConfiguration
	var collectorQuayImages config.ReloadCollectorConfiguration

	config.Load(*configFile, &conf, func() {
		collectorGitHubReleases.ReloadConfiguration(&conf)
		collectorDockerImages.ReloadConfiguration(&conf)
		collectorGitHubTags.ReloadConfiguration(&conf)
		collectorQuayImages.ReloadConfiguration(&conf)

		logger.Debug("flushing cache...")
		cacheClient.Flush()
	})

	collectorGitHubReleases = githubrelease.Register("", conf.GithubReleases, cacheClient)
	collectorDockerImages = dockerimage.Register(conf.DockerImages, cacheClient)
	collectorGitHubTags = githubtag.Register(conf.GithubTags, cacheClient)
	collectorQuayImages = quayimage.Register(conf.QuaryImages, cacheClient)

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(
			w, `
			<html>
			<head><title>Up-to-date Exporter</title></head>
			<body>
				<h1>Up-to-date Exporter</h1>
				<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>
			`,
		)
	})
	logger.Info(fmt.Sprintf(`listening on %s`, *bind))
	if err := http.ListenAndServe(*bind, nil); err != nil { //nolint:gosec
		logger.Error(fmt.Sprintf("error starting server: %s", err))
		panic(err)
	}
}
