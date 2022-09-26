package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh"
	"golang.org/x/exp/slices"
)

const (
	OK = iota
	NG

	tagSeparator = "/"
)

type Flags struct {
	Tag            string
	LabelInclusive string
}

func commitsToPulls(client *Client, commits []*Commit) ([]*Pull, error) {
	pulls := []*Pull{}

	for _, c := range commits {
		commitPulls, err := client.Commits(c.SHA).Pulls()
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

func realMain(releaseTag, labelInclusive string) int {
	repo, err := gh.CurrentRepository()
	if err != nil {
		fmt.Println(err)
		return NG
	}

	restClient, err := gh.RESTClient(nil)
	if err != nil {
		fmt.Println(err)
		return NG
	}

	client := NewClient(restClient, repo.Owner(), repo.Name())

	release, err := client.Releases().Tags(releaseTag)
	if err != nil {
		if !IsNotFound(err) {
			fmt.Println(err)
			return NG
		} else {
			req := &ReleaseCreateRequest{
				Name:                   releaseTag,
				TagName:                releaseTag,
				Body:                   "",
				Draft:                  false,
				Prerelease:             false,
				GenerateReleaseNotes:   false,
				DiscussionCategoryName: nil,
			}
			release, err = client.Releases().Create(req)
			if err != nil {
				fmt.Println(err)
				return NG
			}
		}
	}

	parts := strings.SplitN(releaseTag, tagSeparator, 2)
	prefix := parts[0]

	prev, err := client.PrevRelease(releaseTag, prefix)
	if err != nil && !IsNotFound(err) {
		fmt.Println(err)
		return NG
	}

	compareSince := prev.TagName
	if compareSince == "" {
		compareSince = fmt.Sprintf("%s@{1990-01-01}", release.TargetCommitish)
	}

	comp, err := client.Compare(compareSince, release.TagName)
	if err != nil {
		fmt.Println(err)
		return NG
	}

	pulls, err := commitsToPulls(client, comp.Commits)
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

	req := &ReleaseUpdateRequest{
		ID:                     release.ID,
		Name:                   releaseTag,
		TagName:                releaseTag,
		Body:                   builder.String(),
		Draft:                  false,
		Prerelease:             false,
		GenerateReleaseNotes:   false,
		DiscussionCategoryName: nil,
	}
	release, err = client.Releases().Update(req)
	if err != nil {
		fmt.Println(err)
		return NG
	}

	return OK
}

func main() {
	var f Flags
	flag.StringVar(&f.Tag, "tag", "", "release tag")
	flag.StringVar(&f.LabelInclusive, "label-inclusive", "", "label in pull request inclusive")
	flag.Parse()

	os.Exit(realMain(f.Tag, f.LabelInclusive))
}
