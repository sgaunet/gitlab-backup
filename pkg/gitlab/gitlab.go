package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

var (
	// ErrHTTPStatusCode is returned when an unexpected HTTP status code is received.
	ErrHTTPStatusCode = errors.New("unexpected HTTP status code")
)

const (
	// GitlabAPIEndpoint is the default GitLab API endpoint.
	GitlabAPIEndpoint = "https://gitlab.com/api/v4"
	
	// DownloadRateLimitIntervalSeconds defines the rate limit interval for download API calls.
	DownloadRateLimitIntervalSeconds = 60
	// DownloadRateLimitBurst defines the burst limit for download API calls.
	DownloadRateLimitBurst          = 1
	// ExportRateLimitIntervalSeconds defines the rate limit interval for export API calls.
	ExportRateLimitIntervalSeconds  = 60
	// ExportRateLimitBurst defines the burst limit for export API calls.
	ExportRateLimitBurst           = 6
)

var log Logger

// Logger interface defines the logging methods used by GitLab service.
type Logger interface {
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Info(msg string, args ...any)
}

// Service provides methods to interact with GitLab API.
type Service struct {
	gitlabAPIEndpoint    string
	token                string
	httpClient           *http.Client
	rateLimitDownloadAPI *rate.Limiter
	rateLimitExportAPI   *rate.Limiter
}

func init() {
	log = slog.New(slog.NewTextHandler(io.Discard, nil))
}

// getNextLink parses the link header and returns the next page url
// gitlab API returns a link header with the next page url
// we need to parse this header to get the next page url.
func getNextLink(linkHeader string) string {
	// linkHeader has the format: url1; rel="first", url2; rel="prev", url3; rel="next", url4; rel="last"
	// so we split the string with the , separator
	// and take the first element
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		// link is formatted like this:
		// <https://gitlab.com/api/v4/groups/1234/projects?page=2&per_page=100>; rel="next"
		// we only need the next page url
		// so we split the string with the ; separator
		// and take the first element
		if strings.Contains(link, "rel=\"next\"") {
			nextPageURL := strings.Split(link, ";")[0]
			// remove the < and > characters
			nextPageURL = strings.Trim(nextPageURL, "<> ")
			return nextPageURL
		}
	}
	return ""
}

// NewGitlabService returns a new Service.
func NewGitlabService() *Service {
	gs := &Service{
		gitlabAPIEndpoint: GitlabAPIEndpoint,
		token:             os.Getenv("GITLAB_TOKEN"),
		httpClient:        &http.Client{},
		// implement rate limiting https://docs.gitlab.com/ee/administration/settings/import_export_rate_limits.html
		rateLimitDownloadAPI: rate.NewLimiter(
			rate.Every(DownloadRateLimitIntervalSeconds*time.Second),
			DownloadRateLimitBurst,
		),
		rateLimitExportAPI: rate.NewLimiter(
			rate.Every(ExportRateLimitIntervalSeconds*time.Second),
			ExportRateLimitBurst,
		),
	}
	return gs
}

// SetLogger sets the logger.
func SetLogger(l Logger) {
	if l != nil {
		log = l
	}
}

// SetGitlabEndpoint sets the Gitlab API endpoint
// default: https://gitlab.com/v4/api/
func (r *Service) SetGitlabEndpoint(gitlabAPIEndpoint string) {
	r.gitlabAPIEndpoint = gitlabAPIEndpoint
}

// SetToken sets the Gitlab API token
// default: GITLAB_TOKEN env variable
func (r *Service) SetToken(token string) {
	if token == "" {
		log.Warn("no token provided")
	}
	r.token = token
}

// SetHTTPClient sets the http client
// default: http.Client{}
func (r *Service) SetHTTPClient(httpClient *http.Client) {
	if httpClient != nil {
		r.httpClient = httpClient
	}
}


// GetGroup returns the gitlab group from the given ID.
func (r *Service) GetGroup(ctx context.Context, groupID int) (Group, error) {
	url := fmt.Sprintf("%s/groups/%d", r.gitlabAPIEndpoint, groupID)
	body, err := r.makeRequest(ctx, url, "group")
	if err != nil {
		return Group{}, err
	}

	var res Group
	if err := json.Unmarshal(body, &res); err != nil {
		return Group{}, fmt.Errorf("error unmarshalling json: %w", err)
	}

	return res, nil
}

// GetProject returns informations of the project that matches the given ID.
func (r *Service) GetProject(ctx context.Context, projectID int) (Project, error) {
	url := fmt.Sprintf("%s/projects/%d", r.gitlabAPIEndpoint, projectID)
	body, err := r.makeRequest(ctx, url, "project")
	if err != nil {
		return Project{}, err
	}

	var res Project
	if err := json.Unmarshal(body, &res); err != nil {
		return Project{}, fmt.Errorf("error unmarshalling json: %w", err)
	}

	return res, nil
}
