package metrics

import (
	"context"
	"fmt"
	"github.com/google/go-github/v38/github"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	workflowBillGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_workflow_usage_seconds",
			Help: "Number of billable seconds used by a specific workflow during the current billing cycle. Any job re-runs are also included in the usage. Only apply to workflows in private repositories that use GitHub-hosted runners.",
		},
		[]string{"repo", "name", "os"},
	)
)

// getBillableFromGithub - return billable information for MACOS, WINDOWS and UBUNTU runners.
func getBillableFromGithub(ctx context.Context, owner string, repoName string, workflow github.Workflow) error {
	resp, _, err := client.Actions.GetWorkflowUsageByID(ctx, owner, repoName, *workflow.ID)
	if err != nil {
		return fmt.Errorf("GetWorkflowUsageByID error for %s: %s\n", repoName, err.Error())
	}

	workflowBillGauge.WithLabelValues(repoName, *workflow.Name, "UBUNTU").Set(float64(resp.GetBillable().Ubuntu.GetTotalMS()) / 1000)

	return nil
}
