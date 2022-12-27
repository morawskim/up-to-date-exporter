package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"net/http"
	"up-to-date-exporter/config"
	"up-to-date-exporter/dockerimage"
	"up-to-date-exporter/githubrelease"
)

const (
	bind = ":9333"
)

func main() {
	var conf = config.Config{}
	config.Load("config.yaml", &conf)

	githubrelease.Register("", conf.GithubReleases)
	dockerimage.Register(conf.DockerImages)
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
	log.Info("listening on ", bind)
	if err := http.ListenAndServe(bind, nil); err != nil {
		log.Fatalf("error starting server: %s", err)
	}
}
