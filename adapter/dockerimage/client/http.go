package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

type response struct {
	Results []Release `json:"results"`
	Next    string    `json:"next"`
}

type DockerHubHTTPClient struct {
}

func NewDockerHubClient() *DockerHubHTTPClient {
	return &DockerHubHTTPClient{}
}

func (d *DockerHubHTTPClient) fetchTags(ctx context.Context, url string) (*response, error) {
	var response response

	req, _ := http.NewRequestWithContext( //nolint: noctx
		ctx,
		http.MethodGet,
		url,
		nil,
	)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get repository releases")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(err, "docker hub responded a non-200 status code")
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "failed to parse the response body")
	}

	return &response, nil
}

func (d *DockerHubHTTPClient) Releases(ctx context.Context, container string) ([]Release, error) {
	response, err := d.fetchTags(
		ctx,
		fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/tags?page_size=100", container),
	)

	if err != nil {
		return nil, err
	}

	if len(response.Next) > 0 {
		response2, err := d.fetchTags(ctx, response.Next)

		if err != nil {
			return nil, err
		}

		lenFirstCall := len(response.Results)
		lenSecondCall := len(response2.Results)

		merge := make([]Release, lenFirstCall, lenFirstCall+lenSecondCall)
		_ = copy(merge, response.Results)
		merge = append(merge, response2.Results...)

		return merge, nil
	}

	return response.Results, nil
}
