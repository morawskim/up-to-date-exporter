package client

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
)

type response struct {
	Results []Release `json:"results"`
}

type DockerHubHttpClient struct {
}

func NewDockerHubClient() *DockerHubHttpClient {
	return &DockerHubHttpClient{}
}

func (d *DockerHubHttpClient) Releases(container string) ([]Release, error) {
	var response response

	req, _ := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/tags?page_size=100", container),
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
	return response.Results, nil
}
