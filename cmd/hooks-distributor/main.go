package main

import (
	"github.com/go-semantic-release/semantic-release/v2/pkg/hooks"
	"github.com/go-semantic-release/semantic-release/v2/pkg/plugin"
	hooksDistributor "github.com/superewald/semantic-release-hooks-distributor/pkg/hooks"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		Hooks: func() hooks.Hooks {
			return &hooksDistributor.Distributor{}
		},
	})
}
