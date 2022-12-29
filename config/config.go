package config

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/yaml.v2"
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
		log.Fatalln("failed to load config: ", err)
	}

	var configCh = make(chan os.Signal, 1)
	signal.Notify(configCh, syscall.SIGHUP)

	go func() {
		for range configCh {
			log.Debug("reloading config...")
			if err := doLoad(file, config); err != nil {
				log.Fatalln("failed to reload config: ", err)
			}
			onReload()
			log.Info("config reloaded...")
		}
	}()
}
