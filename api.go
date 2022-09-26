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
)

type Release struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
}

type Commit struct {
	SHA string `json:"sha"`
}

type Compare struct {
	Commits []*Commit `json:"commits"`
}

type User struct {
	Login string `json:"login"`
	URL   string `json:"url"`
}

type Label struct {
	Name string `json:"name"`
}

type Pull struct {
	Number  int64   `json:"number"`
	Title   string  `json:"title"`
	Labels  []Label `json:"labels"`
	HTMLURL string  `json:"html_url"`

	*User `json:"user"`
}

type ReleaseRequest struct {
	ID                     int64
	Name                   string  `json:"name"`
	TagName                string  `json:"tag_name"`
	TargetCommitish        string  `json:"target_commitish"`
	Body                   string  `json:"body"`
	Draft                  bool    `json:"draft"`
	Prerelease             bool    `json:"prerelease"`
	GenerateReleaseNotes   bool    `json:"generate_release_notes"`
	DiscussionCategoryName *string `json:"discussion_category_name,omit_empty"`
}

type ReleaseCreateRequest ReleaseRequest
type ReleaseUpdateRequest ReleaseRequest

type Tag struct {
	Tag string `json:"tag"`
}

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

func (c *releaseClient) Create(req *ReleaseCreateRequest) (*Release, error) {
	path := fmt.Sprintf("repos/%s/%s/releases", c.Owner, c.Repo)
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var release Release
	err = c.RESTClient.Post(path, bytes.NewBuffer(b), &release)
	return &release, err
}

func (c *releaseClient) Update(req *ReleaseUpdateRequest) (*Release, error) {
	path := fmt.Sprintf("repos/%s/%s/releases/%d", c.Owner, c.Repo, req.ID)
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var release Release
	err = c.RESTClient.Post(path, bytes.NewBuffer(b), &release)
	return &release, err
}

func (c *releaseClient) Tags(tag string) (*Release, error) {
	path := fmt.Sprintf("repos/%s/%s/releases/tags/%s", c.Owner, c.Repo, tag)
	var release Release
	err := c.RESTClient.Get(path, &release)
	return &release, err
}

func (c *Client) PrevRelease(tag, prefix string) (*Release, error) {
	page := int(1)
	per_page := int(30)
	for {
		path := fmt.Sprintf("repos/%s/%s/releases?per_page=%d&page=%d", c.Owner, c.Repo, per_page, page)
		var releases []Release
		err := c.RESTClient.Get(path, &releases)
		if err != nil {
			return nil, err
		}
		if len(releases) == 0 {
			break
		}
		for _, r := range releases {
			if r.TagName == tag {
				continue
			}
			if strings.HasPrefix(r.TagName, prefix) {
				return &r, nil
			}
		}
		page++
	}
	// not found and return empty release object.
	return &Release{}, nil
}

func (c *Client) Compare(prevTag, newTag string) (*Compare, error) {
	path := fmt.Sprintf("repos/%s/%s/compare/%s...%s", c.Owner, c.Repo, prevTag, newTag)
	var compare Compare
	err := c.RESTClient.Get(path, &compare)
	return &compare, err
}

func (c *commitClient) Pulls() ([]*Pull, error) {
	path := fmt.Sprintf("repos/%s/%s/commits/%s/pulls", c.Owner, c.Repo, c.Commit)
	var pulls []*Pull
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
