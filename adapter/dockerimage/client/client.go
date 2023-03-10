package client

import (
	"time"
)

type Release struct {
	Tag      string    `json:"name,omitempty"`
	PushedAt time.Time `json:"tag_last_pushed,omitempty"`
}

type DockerHubClient interface {
	Releases(container string) ([]Release, error)
}
