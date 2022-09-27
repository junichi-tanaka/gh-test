package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	_ "github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/google/go-github/v45/github"
)

type Client struct {
	api.RESTClient
	Owner string
	Repo  string
}

type commitClient struct {
	*Client
	Commit string
}

type releaseClient struct {
	*Client
}

func NewClient(client api.RESTClient, owner, repo string) *Client {
	return &Client{
		RESTClient: client,
		Owner:      owner,
		Repo:       repo,
	}
}

func (c *Client) Commits(commit string) *commitClient {
	return &commitClient{
		Client: c,
		Commit: commit,
	}
}

func (c *Client) Releases() *releaseClient {
	return &releaseClient{
		Client: c,
	}
}

func (c *releaseClient) Create(req *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	path := fmt.Sprintf("repos/%s/%s/releases", c.Owner, c.Repo)
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var release github.RepositoryRelease
	err = c.RESTClient.Post(path, bytes.NewBuffer(b), &release)
	return &release, err
}

func (c *releaseClient) Update(req *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	path := fmt.Sprintf("repos/%s/%s/releases/%d", c.Owner, c.Repo, req.ID)
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var release github.RepositoryRelease
	err = c.RESTClient.Post(path, bytes.NewBuffer(b), &release)
	return &release, err
}

func (c *releaseClient) Tags(tag string) (*github.RepositoryRelease, error) {
	path := fmt.Sprintf("repos/%s/%s/releases/tags/%s", c.Owner, c.Repo, tag)
	var release github.RepositoryRelease
	err := c.RESTClient.Get(path, &release)
	return &release, err
}

func (c *Client) PrevRelease(tag, prefix string) (*github.RepositoryRelease, error) {
	page := int(1)
	per_page := int(30)
	for {
		path := fmt.Sprintf("repos/%s/%s/releases?per_page=%d&page=%d", c.Owner, c.Repo, per_page, page)
		var releases []github.RepositoryRelease
		err := c.RESTClient.Get(path, &releases)
		if err != nil {
			return nil, err
		}
		if len(releases) == 0 {
			break
		}
		for _, r := range releases {
			if r.GetTagName() == tag {
				continue
			}
			if strings.HasPrefix(r.GetTagName(), prefix) {
				return &r, nil
			}
		}
		page++
	}
	// not found and return empty release object.
	return &github.RepositoryRelease{}, nil
}

func (c *Client) Compare(prevTag, newTag string) (*github.CommitsComparison, error) {
	path := fmt.Sprintf("repos/%s/%s/compare/%s...%s", c.Owner, c.Repo, prevTag, newTag)
	var compare github.CommitsComparison
	err := c.RESTClient.Get(path, &compare)
	return &compare, err
}

func (c *commitClient) Pulls() ([]*github.PullRequest, error) {
	path := fmt.Sprintf("repos/%s/%s/commits/%s/pulls", c.Owner, c.Repo, c.Commit)
	var pulls []*github.PullRequest
	err := c.RESTClient.Get(path, &pulls)
	return pulls, err
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var httpError api.HTTPError
	if errors.As(err, &httpError) {
		if httpError.StatusCode == http.StatusNotFound {
			return true
		}
	}

	return false
}
