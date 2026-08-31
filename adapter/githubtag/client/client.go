package client

import "context"

type GithubTag struct {
	Tag string `json:"name"`
}

type GithubTagClient interface {
	GetTags(ctx context.Context, repository string) ([]GithubTag, error)
}
