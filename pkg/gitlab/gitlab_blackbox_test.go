package gitlab_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
)

func TestGitlabService_GetGroupGetID(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := gitlab.GitlabGroup{
				Id:   1,
				Name: "test",
			}
			resJSON, err := json.Marshal(res)
			if err != nil {
				t.Errorf("expected no error, got %s", err.Error())
			}
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintln(w, string(resJSON))
		}))
	defer ts.Close()

	// get request
	client := ts.Client()

	r := gitlab.NewGitlabService()
	r.SetHttpClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	g, err := r.GetGroup(1)
	if err != nil {
		t.Errorf("expected no error, got %s", err.Error())
	}
	if g.Id != 1 {
		t.Errorf("expected id %d, got %d", 1, g.Id)
	}
}

func TestGitlabService_GetProjectGetID(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := gitlab.GitlabProject{
				Id:           1,
				Name:         "test",
				Archived:     true,
				ExportStatus: "finished",
			}
			resJSON, err := json.Marshal(res)
			if err != nil {
				t.Errorf("expected no error, got %s", err.Error())
			}
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprintln(w, string(resJSON))
		}))
	defer ts.Close()

	// get request
	client := ts.Client()

	r := gitlab.NewGitlabService()
	r.SetHttpClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	p, err := r.GetProject(1)
	if err != nil {
		t.Errorf("expected no error, got %s", err.Error())
	}
	if p.Id != 1 {
		t.Errorf("expected id %d, got %d", 1, p.Id)
	}
	if p.Name != "test" {
		t.Errorf("expected name %s, got %s", "test", p.Name)
	}
	if p.Archived != true {
		t.Errorf("expected archived %t, got %t", true, p.Archived)
	}
	if p.ExportStatus != "finished" {
		t.Errorf("expected export status %s, got %s", "finished", p.ExportStatus)
	}
}

func TestSetLogger(t *testing.T) {
	gitlab.SetLogger(nil)
	r := gitlab.NewGitlabService()
	r.SetToken("")
}
