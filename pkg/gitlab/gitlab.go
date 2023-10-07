package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

const GitlabApiEndpoint = "https://gitlab.com/api/v4"

type Logger interface {
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Info(msg string, args ...any)
}

type GitlabService struct {
	log               Logger
	gitlabApiEndpoint string
	token             string
	httpClient        *http.Client
}

// NewRequest returns a new GitlabService
func NewService() *GitlabService {
	return &GitlabService{
		gitlabApiEndpoint: GitlabApiEndpoint,
		token:             os.Getenv("GITLAB_TOKEN"),
		httpClient:        &http.Client{},
		log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func (r *GitlabService) SetLogger(log Logger) {
	r.log = log
}

// SetGitlabEndpoint sets the Gitlab API endpoint
// default: https://gitlab.com/v4/api/
func (r *GitlabService) SetGitlabEndpoint(gitlabApiEndpoint string) {
	r.gitlabApiEndpoint = gitlabApiEndpoint
}

// SetToken sets the Gitlab API token
// default: GITLAB_TOKEN env variable
func (r *GitlabService) SetToken(token string) {
	r.token = token
}

// SetHttpClient sets the http client
// default: http.Client{}
func (r *GitlabService) SetHttpClient(httpClient *http.Client) {
	r.httpClient = httpClient
}

// Get sends a GET request to the Gitlab API to the given path
func (r *GitlabService) Get(path string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", r.gitlabApiEndpoint, path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", r.token)
	req.Header.Set("Content-Type", "application/json")
	return r.httpClient.Do(req)
}

// Post sends a POST request to the Gitlab API to the given path
func (r *GitlabService) Post(path string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", r.gitlabApiEndpoint, path)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", r.token)
	return r.httpClient.Do(req)
}

// GetGroup returns the gitlab group from the given ID
func (s *GitlabService) GetGroup(groupID int) (res GitlabGroup, err error) {
	url := fmt.Sprintf("groups/%d", groupID)
	resp, err := s.Get(url)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return res, err
	}
	return res, err
}
