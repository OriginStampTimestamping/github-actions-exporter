package metrics

import (
	"context"
	"fmt"
	"github-actions-exporter/pkg/config"
	"github.com/google/go-github/v38/github"
	"log"
	"strconv"
	"strings"
)

// getFieldValue return value from run element which corresponds to field
func getFieldValue(repo string, workflow github.Workflow, run github.WorkflowRun, field string) string {
	switch field {
	case "repo":
		return repo
	case "id":
		return strconv.FormatInt(*run.ID, 10)
	case "node_id":
		return *run.NodeID
	case "head_branch":
		return *run.HeadBranch
	case "head_sha":
		return *run.HeadSHA
	case "run_number":
		return strconv.Itoa(*run.RunNumber)
	case "workflow_id":
		return strconv.FormatInt(*run.WorkflowID, 10)
	case "workflow":
		return *workflow.Name
	case "event":
		return *run.Event
	case "status":
		return *run.Status
	}
	return ""
}

//
func getRelevantFields(repo string, workflow github.Workflow, run *github.WorkflowRun) []string {
	relevantFields := strings.Split(config.WorkflowFields, ",")
	result := make([]string, len(relevantFields))
	for i, field := range relevantFields {
		result[i] = getFieldValue(repo, workflow, *run, field)
	}
	return result
}

func getWorkflowRunsFromGithub(ctx context.Context, owner string, repoName string, workflow github.Workflow) error {
	opts := &github.ListWorkflowRunsOptions{ListOptions: github.ListOptions{PerPage: 1}}
	log.Printf("Getting runs for %s in %s/%s...", *workflow.Name, owner, repoName)
	resp, _, err := client.Actions.ListWorkflowRunsByID(ctx, owner, repoName, *workflow.ID, opts)
	if err != nil {
		return fmt.Errorf("ListWorkflowRunsByID error for %s and %d: %s", repoName, *workflow.ID, err.Error())
	}

	if len(resp.WorkflowRuns) != 1 {
		log.Printf("  Workflow runs for %s: %d", *workflow.Name, len(resp.WorkflowRuns))
		return nil
	}

	run := resp.WorkflowRuns[0]

	status := 0
	switch run.GetStatus() {
	case "queued":
		status = 1
	case "in_progress":
		status = 2
	case "completed":
		status = 3
	}

	conclusion := 0
	switch run.GetConclusion() {
	case "neutral":
		conclusion = 1
	case "success":
		conclusion = 2
	case "skipped":
		conclusion = 3
	case "cancelled":
		conclusion = 4
	case "timed_out":
		conclusion = 5
	case "action_required":
		conclusion = 6
	case "failure":
		conclusion = 7
	}
	// 37 -> completed failure
	// 32 -> completed success
	value, err := strconv.Atoi(fmt.Sprintf("%d%d", status, conclusion))
	if err != nil {
		return err
	}

	fields := getRelevantFields(repoName, workflow, run)
	workflowRunStatusGauge.WithLabelValues(fields...).Set(float64(value))

	return nil
}
