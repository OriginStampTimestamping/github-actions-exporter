package metrics

import (
	"context"
	"github-actions-exporter/pkg/config"
	"github.com/google/go-github/v38/github"
	"log"
	"time"
)

func runForWorkflow(ctx context.Context, workflowFunc func(ctx context.Context, owner string, repoName string, workflow github.Workflow) error) {
	for range time.Tick(config.Github.Refresh) {
		for _, repo := range config.Github.Repositories.Value() {
			workflowMapLk.RLock()
			workflows, found := workflowMap[repo]
			if !found {
				log.Println("Repo not found in workflow map", repo)
				continue
			}
			for _, v := range workflows {
				time.Sleep(time.Second)
				owner, repoName, ok := config.ParseRepositoryString(repo)
				if !ok {
					continue
				}
				if err = workflowFunc(ctx, owner, repoName, v); err != nil {
					log.Println(err.Error())
				}
			}
			workflowMapLk.RUnlock()
		}
	}
}
