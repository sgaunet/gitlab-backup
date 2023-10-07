package gitlab_test

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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

	r := gitlab.NewService()
	r.SetHttpClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	resp, err := r.Get("groups")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Fatal(err)
	// }
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

	r := gitlab.NewService()
	r.SetHttpClient(client)
	r.SetGitlabEndpoint(ts.URL)
	r.SetToken("test")
	// retrieve groups
	resp, err := r.Get("groups")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
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

	r := gitlab.NewService()
	r.SetHttpClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	resp, err := r.Get("groups")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
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

	r := gitlab.NewService()
	r.SetHttpClient(client)
	r.SetGitlabEndpoint(ts.URL)
	// retrieve groups
	resp, err := r.Post("groups")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
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

	r := gitlab.NewService()
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

	r := gitlab.NewService()
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

	r := gitlab.NewService()
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
