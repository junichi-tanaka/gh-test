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
	Sha string `json:"sha"`
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
	Owner                  string
	Repo                   string
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

type CreateTagRequest struct {
	Owner   string
	Repo    string
	Tag     string `json:"tag"`
	Message string `json:"message"`
	Object  string `json:"object"`
	Type    string `json:"type"`
	Draft   bool   `json:"draft"`
}

type Tag struct {
	Tag string `json:"tag"`
	Sha string `json:"sha"`
}

func GetLatestCommit(client api.RESTClient, owner, repo string) (*Commit, error) {
	path := fmt.Sprintf("repos/%s/%s/commits?page=1&per_page=1", owner, repo)
	commits := []Commit{}
	err := client.Get(path, &commits)
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, nil
	}
	return &commits[0], nil
}

func CreateTag(client api.RESTClient, req *CreateTagRequest) (*Tag, error) {
	path := fmt.Sprintf("repos/%s/%s/git/tags", req.Owner, req.Repo)
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	tag := Tag{}
	err = client.Post(path, bytes.NewBuffer(b), &tag)
	return &tag, err
}

func CreateRelease(client api.RESTClient, req *ReleaseRequest) (*Release, error) {
	path := fmt.Sprintf("repos/%s/%s/releases", req.Owner, req.Repo)
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	release := Release{}
	err = client.Post(path, bytes.NewBuffer(b), &release)
	return &release, err
}

func UpdateRelease(client api.RESTClient, req *ReleaseRequest) (*Release, error) {
	path := fmt.Sprintf("repos/%s/%s/releases/%d", req.Owner, req.Repo, req.ID)
    fmt.Println(path)
    fmt.Println(req)
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	release := Release{}
	err = client.Post(path, bytes.NewBuffer(b), &release)
	return &release, err
}

func GetRelease(client api.RESTClient, owner, repo, tag string) (*Release, error) {
	path := fmt.Sprintf("repos/%s/%s/releases/tags/%s", owner, repo, tag)
	release := &Release{}
	err := client.Get(path, release)
	return release, err
}

func GetPrevRelease(client api.RESTClient, owner, repo, tag string) (*Release, error) {
	const seperator = "/"
	parts := strings.SplitN(tag, seperator, 2)
	prefix := ""
	if len(parts) > 1 {
		prefix = parts[0] + seperator
	}
	page := int(1)
	per_page := int(30)
	for {
		path := fmt.Sprintf("repos/%s/%s/releases?per_page=%d&page=%d", owner, repo, per_page, page)
		releases := []Release{}
		err := client.Get(path, &releases)
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
	return &Release{}, nil
}

func GetCompare(client api.RESTClient, owner, repo, prevTag, newTag string) (*Compare, error) {
	path := fmt.Sprintf("repos/%s/%s/compare/%s...%s", owner, repo, prevTag, newTag)
	comp := Compare{}
	err := client.Get(path, &comp)
	return &comp, err
}

func GetCommitPulls(client api.RESTClient, owner, repo, commit string) ([]*Pull, error) {
	path := fmt.Sprintf("repos/%s/%s/commits/%s/pulls", owner, repo, commit)
	pulls := []*Pull{}
	err := client.Get(path, &pulls)
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
