package config

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

type ReloadCollectorConfiguration interface {
	prometheus.Collector
	ReloadConfiguration(config *Config)
}

type Config struct {
	GithubReleases map[string]string `yaml:"github_releases"`
	DockerImages   map[string]string `yaml:"docker_images"`
	GithubTags     map[string]string `yaml:"github_tags"`
}

func doLoad(file string, config *Config) error {
	bts, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var newConfig Config
	if err := yaml.Unmarshal(bts, &newConfig); err != nil {
		return err
	}
	*config = newConfig

	return nil
}

func Load(file string, config *Config, onReload func()) {
	if err := doLoad(file, config); err != nil {
		slog.Default().Error(fmt.Sprintf("failed to load config: %s", err))
		panic(err)
	}

	var configCh = make(chan os.Signal, 1)
	signal.Notify(configCh, syscall.SIGHUP)

	go func() {
		for range configCh {
			slog.Default().Debug("reloading config...")
			if err := doLoad(file, config); err != nil {
				slog.Default().Error(fmt.Sprintf("failed to reload config: %s", err))
				panic(err)
			}
			onReload()
			slog.Default().Info("config reloaded...")
		}
	}()
}
