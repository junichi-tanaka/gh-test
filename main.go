package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"golang.org/x/exp/slices"
)

const (
	OK = iota
	NG
)

func commitsToPulls(client api.RESTClient, owner, repo string, commits []*Commit) ([]*Pull, error) {
	pulls := []*Pull{}

	for _, c := range commits {
		commitPulls, err := GetCommitPulls(client, owner, repo, c.Sha)
		if err != nil {
			return nil, err
		}
		for _, p := range commitPulls {
			pulls = append(pulls, p)
		}
	}

	return pulls, nil
}

func filterPulls(pulls []*Pull, labelInclusive string) []*Pull {
	filtered := []*Pull{}

	for _, p := range pulls {
		if slices.Contains(p.Labels, Label{Name: labelInclusive}) {
			filtered = append(filtered, p)
		}
	}

	return filtered
}

func realMain(owner, repo, releaseTag, labelInclusive string) int {
	client, err := gh.RESTClient(nil)
	if err != nil {
		fmt.Println(err)
		return NG
	}

	release, err := GetRelease(client, owner, repo, releaseTag)
	if err != nil {
		if !IsNotFound(err) {
			fmt.Println(err)
			return NG
		} else {
			req := &ReleaseRequest{
				Owner:                  owner,
				Repo:                   repo,
				Name:                   releaseTag,
				TagName:                releaseTag,
				Body:                   "",
				Draft:                  false,
				Prerelease:             false,
				GenerateReleaseNotes:   false,
				DiscussionCategoryName: nil,
			}
			release, err = CreateRelease(client, req)
			if err != nil {
				fmt.Println(err)
				return NG
			}
		}
	}

	prev, err := GetPrevRelease(client, owner, repo, releaseTag)
	if err != nil && !IsNotFound(err) {
		fmt.Println(err)
		return NG
	}

    compareSince := prev.TagName
    if compareSince == "" {
        compareSince = fmt.Sprintf("%s@{1990-01-01}", release.TargetCommitish)
    }

	comp, err := GetCompare(client, owner, repo, compareSince, release.TagName)
	if err != nil {
		fmt.Println(err)
		return NG
	}

	pulls, err := commitsToPulls(client, owner, repo, comp.Commits)
	if err != nil {
		fmt.Println(err)
		return NG
	}

	pulls = filterPulls(pulls, labelInclusive)

	var builder strings.Builder
	for _, p := range pulls {
		line := fmt.Sprintf("- %s by @%s in %s\n", p.Title, p.User.Login, p.HTMLURL)
		builder.WriteString(line)
	}

	req := &ReleaseRequest{
		Owner:                  owner,
		Repo:                   repo,
		ID:                     release.ID,
		Name:                   releaseTag,
		TagName:                releaseTag,
		Body:                   builder.String(),
		Draft:                  false,
		Prerelease:             false,
		GenerateReleaseNotes:   false,
		DiscussionCategoryName: nil,
	}
	release, err = UpdateRelease(client, req)
	if err != nil {
		fmt.Println(err)
		return NG
	}

	return OK
}

func main() {
	owner := "owner name of the reposigory"
	repo := "the target repository name"
	releaseTag := "the tag name for the next release"
	labelInclusive := "label of the pull request to be included in the release note"

	os.Exit(realMain(owner, repo, releaseTag, labelInclusive))
}
