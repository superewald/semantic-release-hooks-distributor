package hooks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-semantic-release/semantic-release/v2/pkg/config"
	"github.com/go-semantic-release/semantic-release/v2/pkg/hooks"
	"github.com/google/go-github/v49/github"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

var HVERSION = "dev"

type AssetFileConfig struct {
	filepath    string
	filename    string
	releasename string
	packagename string
}

type Distributor struct {
	providerName string
	providerOpts map[string]string
	projectID    string
	assetFiles   []AssetFileConfig
}

func (dst *Distributor) Init(opts map[string]string) error {
	cmd := &cobra.Command{}
	config.SetFlags(cmd)
	config.InitConfig(cmd)
	semrelConf, err := config.NewConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to retrieve new config: %w", err)
	}

	dst.providerOpts = semrelConf.ProviderOpts
	dst.providerName = opts["provider"]

	assets := opts["assets"]
	if assets == "" {
		assets = os.Getenv("SEMREL_ASSETS")
	}
	if assets == "" {
		return fmt.Errorf("asset file list missing")
	}

	r := regexp.MustCompile(`([^:]+){1}\:?([^@]+)?\@?(\w+)?`)
	assetFiles := strings.Split(opts["assets"], " ")
	for _, asset := range assetFiles {
		groups := r.FindStringSubmatch(asset)[1:]
		searchGlob := groups[0]
		searchRegexString := strings.ReplaceAll(searchGlob, "?", "(.*)")
		searchRegexString = strings.ReplaceAll(searchRegexString, "*", `(\w*)`)
		searchRegexString = strings.ReplaceAll(searchRegexString, "[", `([`)
		searchRegexString = strings.ReplaceAll(searchRegexString, "]", "]+)")
		searchRegex := regexp.MustCompile(searchRegexString)

		matches, err := filepath.Glob(searchGlob)
		if err != nil {
			return fmt.Errorf("failed to resolve glob: %w", err)
		}

		for _, match := range matches {
			releaseName := filepath.Base(match)
			if groups[1] != "" {
				releaseName = searchRegex.ReplaceAllString(match, groups[1])
			}

			assetFileConf := AssetFileConfig{
				filepath:    match,
				filename:    filepath.Base(match),
				releasename: releaseName,
				packagename: groups[2],
			}
			dst.assetFiles = append(dst.assetFiles, assetFileConf)
		}
	}

	return nil
}

func (dst *Distributor) Success(shConfig *hooks.SuccessHookConfig) error {
	if dst.providerName == "GitHub" {
		client, err := newGithubClient(dst.providerOpts)
		if err != nil {
			return fmt.Errorf("failed to create github client: %w", err)
		}

		latestRelease, _, err := client.Repositories.GetLatestRelease(context.Background(), shConfig.RepoInfo.Owner, shConfig.RepoInfo.Repo)
		if err != nil {
			return errors.New("failed to get latest release: " + err.Error())
		}

		for _, assetFile := range dst.assetFiles {
			f, err := os.Open(assetFile.filepath)
			if err != nil {
				log.Default().Printf("failed to upload release asset: %s", err)
			}
			_, _, err = client.Repositories.UploadReleaseAsset(context.Background(), shConfig.RepoInfo.Owner, shConfig.RepoInfo.Repo, *latestRelease.ID, &github.UploadOptions{}, f)
			if err != nil {
				log.Default().Printf("failed to upload release asset: %s", err)
			}
			f.Close()
		}
	} else if dst.providerName == "GitLab" {
		client, err := newGitlabClient(dst.providerOpts)
		if err != nil {
			return fmt.Errorf("failed to create gitlab client: %w", err)
		}

		for _, assetFile := range dst.assetFiles {
			f, err := os.Open(assetFile.filepath)
			if err != nil {
				log.Default().Printf("failed to upload package asset: %s", err)
			}

			packageName := assetFile.packagename
			if packageName == "" {
				packageName = shConfig.RepoInfo.Repo
			}

			pkgfile, _, err := client.GenericPackages.PublishPackageFile(dst.providerOpts["gitlab_projectid"], packageName, shConfig.NewRelease.Version, assetFile.releasename, f, &gitlab.PublishPackageFileOptions{
				Select: gitlab.GenericPackageSelect(gitlab.SelectPackageFile),
			})
			if err != nil {
				log.Printf("failed to upload package asset: %s", err)
			}
			f.Close()

			_, _, err = client.ReleaseLinks.CreateReleaseLink(dst.providerOpts["gitlab_projectid"], "v"+shConfig.NewRelease.Version, &gitlab.CreateReleaseLinkOptions{
				Name:     &assetFile.releasename,
				URL:      &pkgfile.File.URL,
				LinkType: gitlab.LinkType(gitlab.OtherLinkType),
			})
			if err != nil {
				log.Default().Printf("failed to create release link: %s", err)
			}
		}
	} else {
		return fmt.Errorf("no provider specified: %s", dst.providerName)
	}

	return nil
}

func (dst *Distributor) NoRelease(config *hooks.NoReleaseConfig) error {
	return nil
}

func (dst *Distributor) Name() string {
	return "Distributor"
}

func (dst *Distributor) Version() string {
	return HVERSION
}
