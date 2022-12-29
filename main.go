package main

import (
	"fmt"
	"github.com/alecthomas/kingpin"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"net/http"
	"time"
	"up-to-date-exporter/adapter/githubtag"
	"up-to-date-exporter/config"
	"up-to-date-exporter/dockerimage"
	"up-to-date-exporter/githubrelease"
)

var (
	bind       = kingpin.Flag("bind", "addr to bind the server").Default(":9333").String()
	debug      = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	configFile = kingpin.Flag("config.file", "config file").Default("config.yaml").ExistingFile()

	version = "dev"
)

func main() {
	kingpin.Version("up-to-date-exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	log.Info("starting up-to-date-exporter")

	if *debug {
		_ = log.Base().SetLevel("debug")
		log.Debug("enabled debug mode")
	}

	cacheClient := cache.New(time.Minute*15, time.Minute*15)

	var conf = config.Config{}
	var collectorGitHubReleases, collectorDockerImages, collectorGitHubTags config.ReloadCollectorConfiguration

	config.Load(*configFile, &conf, func() {
		collectorGitHubReleases.ReloadConfiguration(&conf)
		collectorDockerImages.ReloadConfiguration(&conf)
		collectorGitHubTags.ReloadConfiguration(&conf)

		log.Debug("flushing cache...")
		cacheClient.Flush()
	})

	collectorGitHubReleases = githubrelease.Register("", conf.GithubReleases, cacheClient)
	collectorDockerImages = dockerimage.Register(conf.DockerImages, cacheClient)
	collectorGitHubTags = githubtag.Register(conf.GithubTags, cacheClient)

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
	log.Info("listening on ", *bind)
	if err := http.ListenAndServe(*bind, nil); err != nil {
		log.Fatalf("error starting server: %s", err)
	}
}
