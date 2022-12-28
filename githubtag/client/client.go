package client

type GithubTag struct {
	Tag string `json:"name"`
}

type GithubTagClient interface {
	GetTags(repository string) ([]GithubTag, error)
}
