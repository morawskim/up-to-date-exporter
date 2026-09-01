package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

type GithubReleaseHTTPClient struct {
	token string
}

func NewGithubReleaseHTTPClient(token string) *GithubReleaseHTTPClient {
	return &GithubReleaseHTTPClient{token: token}
}

func (c *GithubReleaseHTTPClient) Releases(ctx context.Context, repository string) ([]GithubRelease, error) {
	var result []GithubRelease
	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/releases", repository),
		nil,
	)

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get repository releases")
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("GitHub responded a non-200 status code")
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to parse the response body")
	}

	return result, nil
}
