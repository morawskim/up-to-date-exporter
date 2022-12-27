package config

import (
	"github.com/prometheus/common/log"
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	GithubReleases map[string]string `yaml:"github_releases"`
	DockerImages   map[string]string `yaml:"docker_images"`
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

func Load(file string, config *Config) {
	if err := doLoad(file, config); err != nil {
		log.Fatalln("failed to load config: ", err)
	}
}
