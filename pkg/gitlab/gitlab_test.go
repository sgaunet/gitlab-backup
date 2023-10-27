package gitlab_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
)

func TestGitlabService_SetGitlabEndpoint(t *testing.T) {
	response := []struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	}{
		{
			Id:   1,
			Name: "test",
		},
	}
	responseJSON, _ := json.Marshal(response)
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, string(responseJSON))
		}))
	defer ts.Close()

	// get request
	client := ts.Client()

	r := gitlab.NewGitlabService()
	r.SetHttpClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	resp, err := r.Get(fmt.Sprintf("%s/groups", ts.URL))
	if err != nil {
		t.Error(err.Error())
	}
	defer resp.Body.Close()
}

func TestGitlabService_CheckTokenInHeader(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// get token from header
			token := r.Header.Get("PRIVATE-TOKEN")
			if token == "" {
				t.Error("token not found in header")
				// return unauthorized
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, string("unauthorized"))
			}
			fmt.Fprintln(w, string("ok"))
		}))
	defer ts.Close()

	// get request
	client := ts.Client()

	r := gitlab.NewGitlabService()
	r.SetHttpClient(client)
	r.SetGitlabEndpoint(ts.URL)
	r.SetToken("test")
	// retrieve groups
	resp, err := r.Get(fmt.Sprintf("%s/groups", ts.URL))
	if err != nil {
		t.Error(err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if string(body) != "ok\n" {
		t.Errorf("expected body %s, got %s", "ok\n", string(body))
	}
}

func TestGitlabService_CheckTokenInHeaderFromEnvVar(t *testing.T) {
	os.Setenv("GITLAB_TOKEN", "test")
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// get token from header
			token := r.Header.Get("PRIVATE-TOKEN")
			if token == "" {
				t.Error("token not found in header")
				// return unauthorized
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, string("unauthorized"))
			}
			if token != "test" {
				t.Errorf("expected token %s, got %s", "test", token)
			}
			fmt.Fprintln(w, string("ok"))
		}))
	defer ts.Close()

	// get request
	client := ts.Client()

	r := gitlab.NewGitlabService()
	r.SetHttpClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	resp, err := r.Get(fmt.Sprintf("%s/groups", ts.URL))
	if err != nil {
		t.Error(err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if string(body) != "ok\n" {
		t.Errorf("expected body %s, got %s", "ok\n", string(body))
	}
}

func TestGitlabService_CheckPost(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// check method post
			if r.Method == "POST" {
				fmt.Fprintln(w, "ok")
			}
		}))
	defer ts.Close()

	// get request
	client := ts.Client()

	r := gitlab.NewGitlabService()
	r.SetHttpClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	resp, err := r.Post(fmt.Sprintf("%s/groups", ts.URL))
	if err != nil {
		t.Error(err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if string(body) != "ok\n" {
		t.Errorf("expected body %s, got %s", "ok\n", string(body))
	}
}

func TestGitlabService_GetGroup(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := gitlab.GitlabGroup{
				Id:   1,
				Name: "test",
			}
			// check method GET
			if r.Method != "GET" {
				t.Errorf("expected method %s, got %s", "GET", r.Method)
			}
			// check path
			if r.URL.Path != "/groups/1" {
				t.Errorf("expected path %s, got %s", "/groups/1", r.URL.Path)
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
	_, err := r.GetGroup(1)
	if err != nil {
		t.Errorf("expected no error, got %s", err.Error())
	}
}

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

func TestGetNextLink(t *testing.T) {
	value := "<https://gitlab.com/api/v4/groups/111/subgroups?id=111&order_by=id&owned=false&page=1&pagination=keyset&per_page=20&sort=asc&statistics=false&with_custom_attributes=false>; rel=\"first\", <https://gitlab.com/api/v4/groups/111/subgroups?id=111&order_by=id&owned=false&page=1&pagination=keyset&per_page=20&sort=asc&statistics=false&with_custom_attributes=false>; rel=\"last\""
	res := gitlab.GetNextLink(value)
	if res != "" {
		t.Error("expected empty string, got ", res)
	}
}

func TestGetNextLink2(t *testing.T) {
	expected := "https://gitlab.com/api/v4/groups/111/projects?id=111&include_ancestor_groups=false&include_subgroups=false&order_by=id&owned=false&page=2&pagination=keyset&per_page=20&simple=false&sort=asc&starred=false&with_custom_attributes=false&with_issues_enabled=false&with_merge_requests_enabled=false&with_security_reports=false&with_shared=true"
	value := "<https://gitlab.com/api/v4/groups/111/projects?id=111&include_ancestor_groups=false&include_subgroups=false&order_by=id&owned=false&page=2&pagination=keyset&per_page=20&simple=false&sort=asc&starred=false&with_custom_attributes=false&with_issues_enabled=false&with_merge_requests_enabled=false&with_security_reports=false&with_shared=true>; rel=\"next\", <https://gitlab.com/api/v4/groups/111/projects?id=111&include_ancestor_groups=false&include_subgroups=false&order_by=id&owned=false&page=1&pagination=keyset&per_page=20&simple=false&sort=asc&starred=false&with_custom_attributes=false&with_issues_enabled=false&with_merge_requests_enabled=false&with_security_reports=false&with_shared=true>; rel=\"first\", <https://gitlab.com/api/v4/groups/111/projects?id=111&include_ancestor_groups=false&include_subgroups=false&order_by=id&owned=false&page=4&pagination=keyset&per_page=20&simple=false&sort=asc&starred=false&with_custom_attributes=false&with_issues_enabled=false&with_merge_requests_enabled=false&with_security_reports=false&with_shared=true>; rel=\"last\""
	res := gitlab.GetNextLink(value)
	if res != "https://gitlab.com/api/v4/groups/111/projects?id=111&include_ancestor_groups=false&include_subgroups=false&order_by=id&owned=false&page=2&pagination=keyset&per_page=20&simple=false&sort=asc&starred=false&with_custom_attributes=false&with_issues_enabled=false&with_merge_requests_enabled=false&with_security_reports=false&with_shared=true" {
		t.Errorf("expected string=%s, got %s", expected, res)
	}
}

func TestSetLogger(t *testing.T) {
	gitlab.SetLogger(nil)
	r := gitlab.NewGitlabService()
	r.SetToken("")
}
