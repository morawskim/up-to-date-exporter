package client

import "context"

type GithubRelease struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

type GithubReleaseClient interface {
	Releases(ctx context.Context, repository string) ([]GithubRelease, error)
}
