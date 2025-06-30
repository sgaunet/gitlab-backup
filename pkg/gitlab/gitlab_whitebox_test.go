package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

func TestGetNextLinkReturnEmptyStrIfNoNextLink(t *testing.T) {
	value := "<https://gitlab.com/api/v4/groups/111/subgroups?id=111&order_by=id&owned=false&page=1&pagination=keyset&per_page=20&sort=asc&statistics=false&with_custom_attributes=false>; rel=\"first\", <https://gitlab.com/api/v4/groups/111/subgroups?id=111&order_by=id&owned=false&page=1&pagination=keyset&per_page=20&sort=asc&statistics=false&with_custom_attributes=false>; rel=\"last\""
	res := getNextLink(value)
	if res != "" {
		t.Error("expected empty string, got ", res)
	}
}

func TestGetNextLinkReturnTheNextLinkIfPresent(t *testing.T) {
	expected := "https://gitlab.com/api/v4/groups/111/projects?id=111&include_ancestor_groups=false&include_subgroups=false&order_by=id&owned=false&page=2&pagination=keyset&per_page=20&simple=false&sort=asc&starred=false&with_custom_attributes=false&with_issues_enabled=false&with_merge_requests_enabled=false&with_security_reports=false&with_shared=true"
	value := "<https://gitlab.com/api/v4/groups/111/projects?id=111&include_ancestor_groups=false&include_subgroups=false&order_by=id&owned=false&page=2&pagination=keyset&per_page=20&simple=false&sort=asc&starred=false&with_custom_attributes=false&with_issues_enabled=false&with_merge_requests_enabled=false&with_security_reports=false&with_shared=true>; rel=\"next\", <https://gitlab.com/api/v4/groups/111/projects?id=111&include_ancestor_groups=false&include_subgroups=false&order_by=id&owned=false&page=1&pagination=keyset&per_page=20&simple=false&sort=asc&starred=false&with_custom_attributes=false&with_issues_enabled=false&with_merge_requests_enabled=false&with_security_reports=false&with_shared=true>; rel=\"first\", <https://gitlab.com/api/v4/groups/111/projects?id=111&include_ancestor_groups=false&include_subgroups=false&order_by=id&owned=false&page=4&pagination=keyset&per_page=20&simple=false&sort=asc&starred=false&with_custom_attributes=false&with_issues_enabled=false&with_merge_requests_enabled=false&with_security_reports=false&with_shared=true>; rel=\"last\""
	res := getNextLink(value)
	if res != "https://gitlab.com/api/v4/groups/111/projects?id=111&include_ancestor_groups=false&include_subgroups=false&order_by=id&owned=false&page=2&pagination=keyset&per_page=20&simple=false&sort=asc&starred=false&with_custom_attributes=false&with_issues_enabled=false&with_merge_requests_enabled=false&with_security_reports=false&with_shared=true" {
		t.Errorf("expected string=%s, got %s", expected, res)
	}
}

func TestGitlabService_GetGroup(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := Group{
				ID:   1,
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

	r := NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	_, err := r.GetGroup(context.Background(), 1)
	if err != nil {
		t.Errorf("expected no error, got %s", err.Error())
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

	r := NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	resp, err := r.get(context.Background(), fmt.Sprintf("%s/groups", ts.URL))
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

func TestGitlabService_SetGitlabEndpoint(t *testing.T) {
	response := []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{
		{
			ID:   1,
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

	r := NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	resp, err := r.get(context.Background(), fmt.Sprintf("%s/groups", ts.URL))
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

	r := NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL)
	r.SetToken("test")
	// retrieve groups
	resp, err := r.get(context.Background(), fmt.Sprintf("%s/groups", ts.URL))
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

func TestGitlabService_askExport_Success(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected method %s, got %s", "POST", r.Method)
			}
			// check path
			if r.URL.Path != "/projects/1/export" {
				t.Errorf("expected path %s, got %s", "/projects/1/export", r.URL.Path)
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
		}))
	defer ts.Close()

	// get request
	client := ts.Client()

	r := NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	res, err := r.askExport(context.Background(), 1)
	if err != nil {
		t.Errorf("expected no error, got %s", err.Error())
	}
	if res != true {
		t.Errorf("expected true, got %v", res)
	}
}

func TestGitlabService_getStatusExport(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var res Project
			res.Archived = false
			res.ExportStatus = "finished"
			res.ID = 1
			res.Name = "test"

			if r.Method != "GET" {
				t.Errorf("expected method %s, got %s", "GET", r.Method)
			}
			// check path
			if r.URL.Path != "/projects/1/export" {
				t.Errorf("expected path %s, got %s", "/projects/1/export", r.URL.Path)
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

	r := NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	res, err := r.getStatusExport(context.Background(), 1)
	if err != nil {
		t.Errorf("expected no error, got %s", err.Error())
	}
	if res != "finished" {
		t.Errorf("expected finished, got %v", res)
	}
}

func TestGitlabService_waitForExport(t *testing.T) {
	exportstatus := ""
	nbrequest := 0
	ts := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var res Project
			res.Archived = false
			res.ExportStatus = exportstatus
			res.ID = 1
			res.Name = "test"
			nbrequest++

			if r.Method != "GET" {
				t.Errorf("expected method %s, got %s", "GET", r.Method)
			}
			// check path
			if r.URL.Path != "/projects/1/export" {
				t.Errorf("expected path %s, got %s", "/projects/1/export", r.URL.Path)
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

	r := NewGitlabService()
	r.SetHTTPClient(client)
	r.SetGitlabEndpoint(ts.URL)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := r.waitForExport(context.Background(), 1)
		if err != nil {
			t.Errorf("expected no error, got %s", err.Error())
		}
		wg.Done()
	}()
	time.Sleep(10 * time.Second)
	exportstatus = "finished"
	wg.Wait()
	if nbrequest != 3 {
		t.Errorf("expected 3 requests, got %d", nbrequest)
	}

}
