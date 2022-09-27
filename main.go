package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh"
	"github.com/google/go-github/v45/github"
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

func commitsToPulls(client *Client, commits []*github.RepositoryCommit) ([]*github.PullRequest, error) {
	var pulls []*github.PullRequest

	for _, c := range commits {
		commitPulls, err := client.Commits(c.GetSHA()).Pulls()
		if err != nil {
			return nil, err
		}
		for _, p := range commitPulls {
			pulls = append(pulls, p)
		}
	}

	return pulls, nil
}

func filterPulls(pulls []*github.PullRequest, labelInclusive string) []*github.PullRequest {
	var filtered []*github.PullRequest

	for _, p := range pulls {
		if slices.IndexFunc(p.Labels, func(l *github.Label) bool {
			if l.GetName() == labelInclusive {
				return true
			}
			return false
		}) >= 0 {
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
			req := &github.RepositoryRelease{
				Name:                 github.String(releaseTag),
				TagName:              github.String(releaseTag),
				Draft:                github.Bool(false),
				Prerelease:           github.Bool(false),
				GenerateReleaseNotes: github.Bool(false),
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

	compareSince := prev.GetTagName()
	if compareSince == "" {
		compareSince = fmt.Sprintf("%s@{1990-01-01}", release.TargetCommitish)
	}

	comp, err := client.Compare(compareSince, release.GetTagName())
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
		line := fmt.Sprintf("- %s by @%s in %s\n", p.GetTitle(), p.GetUser().GetLogin(), p.GetHTMLURL())
		builder.WriteString(line)
	}

	req := &github.RepositoryRelease{
		ID:                   github.Int64(release.GetID()),
		Name:                 github.String(releaseTag),
		TagName:              github.String(releaseTag),
		Body:                 github.String(builder.String()),
		Draft:                github.Bool(false),
		Prerelease:           github.Bool(false),
		GenerateReleaseNotes: github.Bool(false),
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
