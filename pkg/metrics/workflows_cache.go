package metrics

import (
	"context"
	"log"
	"sync"
	"time"

	"github-actions-exporter/pkg/config"
	"github.com/google/go-github/v38/github"
)

var (
	workflowMapLk sync.RWMutex
	workflowMap   map[string]map[int64]github.Workflow
)

// workflowCache - used for limit calls to github api
func workflowCache(ctx context.Context) {
	for range time.Tick(5 * time.Minute) {
		workflowsByRepo := getWorkflows(ctx)
		workflowMapLk.Lock()
		workflowMap = workflowsByRepo
		workflowMapLk.Unlock()
	}
}

func getWorkflows(ctx context.Context) map[string]map[int64]github.Workflow {
	log.Println("Getting a list of all configured workflows")

	// map of "owner/repo" -> Workflow ID -> Workflow
	workflowsByRepo := make(map[string]map[int64]github.Workflow)

	for _, repo := range config.Github.Repositories.Value() {
		owner, repoName, ok := config.ParseRepositoryString(repo)
		if !ok {
			log.Println("Could not ListWorkflows as repository config does not provide owner and repo separated by a '/':", repo)
			continue
		}

		log.Println("Getting a list of workflows for", repo)
		opts := &github.ListOptions{Page: 0, PerPage: 1000}
		resp, _, err := client.Actions.ListWorkflows(ctx, owner, repoName, opts)
		if err != nil {
			log.Printf("ListWorkflows error for %s: %s\n", repo, err.Error())
			continue
		}

		s := make(map[int64]github.Workflow)
		for _, w := range resp.Workflows {
			if w == nil {
				log.Println("Workflow was null in repo", repo)
				continue
			}
			s[*w.ID] = *w
		}

		workflowsByRepo[repo] = s
	}

	return workflowsByRepo
}
