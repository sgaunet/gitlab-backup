// gitlab-backup
// Copyright (C) 2021  Sylvain Gaunet

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package gitlabProject

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type GitlabProject struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func New(projectID int) (res GitlabProject, err error) {
	url := fmt.Sprintf("%s/api/v4/projects/%d", os.Getenv("GITLAB_URI"), projectID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return res, err
	}
	req.Header.Set("PRIVATE-TOKEN", os.Getenv("GITLAB_TOKEN"))
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return res, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return res, err
	}

	return res, err
}
