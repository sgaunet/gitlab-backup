package gitlabRequest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

const GitlabApiVersion = "v4"

func Request(uri string) (resp *http.Response, body []byte, err error) {
	url := fmt.Sprintf("%s/api/%s/%s", os.Getenv("GITLAB_URI"), GitlabApiVersion, uri)
	log.Debugf("url: %s\n", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", os.Getenv("GITLAB_TOKEN"))
	client := &http.Client{}
	resp, err = client.Do(req)

	if err != nil {
		return nil, nil, err
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	log.Debugf("body: %v\n", string(body))

	return resp, body, err
}
