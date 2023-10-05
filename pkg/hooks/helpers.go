package hooks

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/google/go-github/v49/github"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

func newGithubClient(config map[string]string) (*github.Client, error) {
	gheHost := config["github_enterprise_host"]
	if gheHost == "" {
		gheHost = os.Getenv("GITHUB_ENTERPRISE_HOST")
	}

	token := config["token"]
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		return nil, errors.New("github token missing")
	}

	oauthClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))

	if gheHost != "" {
		gheURL := fmt.Sprintf("https://%s/api/v3/", gheHost)
		return github.NewEnterpriseClient(gheURL, gheURL, oauthClient)
	} else {
		return github.NewClient(oauthClient), nil
	}
}

func newGitlabClient(config map[string]string) (*gitlab.Client, error) {
	gitlabBaseURl := config["gitlab_baseurl"]
	if gitlabBaseURl == "" {
		gitlabBaseURl = os.Getenv("CI_SERVER_URL")
	}

	useJobToken := false
	token := config["token"]
	if token == "" {
		token = os.Getenv("GITLAB_TOKEN")
	}
	if token == "" {
		token = os.Getenv("CI_JOB_TOKEN")
		useJobToken = true
	}
	if token == "" {
		return nil, errors.New("gitlab token missing")
	}

	gitlabClientOpts := []gitlab.ClientOptionFunc{}

	if gitlabBaseURl != "" {
		gitlabClientOpts = append(gitlabClientOpts, gitlab.WithBaseURL(gitlabBaseURl))
	}

	var client *gitlab.Client
	var err error

	if useJobToken {
		client, err = gitlab.NewJobClient(token, gitlabClientOpts...)
	} else {
		client, err = gitlab.NewClient(token, gitlabClientOpts...)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}
