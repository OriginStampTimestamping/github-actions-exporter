package metrics

import (
	"context"
	"fmt"
	"github-actions-exporter/pkg/config"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v38/github"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/oauth2"
)

var (
	client                   *github.Client
	err                      error
	workflowRunStatusGauge   *prometheus.GaugeVec
)

// InitMetrics - register metrics in prometheus lib and start func for monitor
func InitMetrics(ctx context.Context) {
	workflowRunStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_workflow_run_status",
			Help: "Workflow run status",
		},
		strings.Split(config.WorkflowFields, ","),
	)
	prometheus.MustRegister(workflowRunStatusGauge)
	prometheus.MustRegister(workflowBillGauge)

	client, err = NewClient(ctx)
	if err != nil {
		log.Fatalln("Error: Client creation failed." + err.Error())
	}

	workflowsByRepo := getWorkflows(ctx)
	workflowMapLk.Lock()
	workflowMap = workflowsByRepo
	workflowMapLk.Unlock()

	go workflowCache(ctx)

	for {
		if workflowMap != nil {
			break
		}
	}

	go runForWorkflow(ctx, getBillableFromGithub)
	go runForWorkflow(ctx, getWorkflowRunsFromGithub)
}

// NewClient creates a Github Client
func NewClient(ctx context.Context) (*github.Client, error) {
	var (
		httpClient *http.Client
		client     *github.Client
		transport  http.RoundTripper
	)
	if len(config.Github.Token) > 0 {
		log.Printf("authenticating with Github Token")
		transport = oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.Github.Token})).Transport
		httpClient = &http.Client{Transport: transport}
	} else {
		log.Printf("authenticating with Github App")
		tr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, config.Github.AppID, config.Github.AppInstallationID, config.Github.AppPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("authentication failed: %v", err)
		}
		if config.Github.APIURL != "api.github.com" {
			githubAPIURL, err := getEnterpriseApiUrl(config.Github.APIURL)
			if err != nil {
				return nil, fmt.Errorf("enterprise url incorrect: %v", err)
			}
			tr.BaseURL = githubAPIURL
		}
		httpClient = &http.Client{Transport: tr}
	}

	if config.Github.APIURL != "api.github.com" {
		var err error
		client, err = github.NewEnterpriseClient(config.Github.APIURL, config.Github.APIURL, httpClient)
		if err != nil {
			return nil, fmt.Errorf("enterprise client creation failed: %v", err)
		}
	} else {
		client = github.NewClient(httpClient)
	}

	return client, nil
}

func getEnterpriseApiUrl(baseURL string) (string, error) {
	baseEndpoint, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(baseEndpoint.Path, "/") {
		baseEndpoint.Path += "/"
	}
	if !strings.HasSuffix(baseEndpoint.Path, "/api/v3/") &&
		!strings.HasPrefix(baseEndpoint.Host, "api.") &&
		!strings.Contains(baseEndpoint.Host, ".api.") {
		baseEndpoint.Path += "api/v3/"
	}

	// Trim trailing slash, otherwise there's double slash added to token endpoint
	return fmt.Sprintf("%s://%s%s", baseEndpoint.Scheme, baseEndpoint.Host, strings.TrimSuffix(baseEndpoint.Path, "/")), nil
}
