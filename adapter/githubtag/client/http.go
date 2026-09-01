package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

type GithubTagsHTTPClient struct {
	token string
}

func NewGithubTagHTTPClient(token string) *GithubTagsHTTPClient {
	return &GithubTagsHTTPClient{
		token: token,
	}
}

func (c *GithubTagsHTTPClient) GetTags(ctx context.Context, repository string) ([]GithubTag, error) {
	var result []GithubTag
	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/tags", repository),
		nil,
	)
	response, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, errors.Wrap(err, "failed to get repository releases")
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Wrap(err, "GitHub responded a non-200 status code")
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to parse the response body")
	}

	return result, nil
}
