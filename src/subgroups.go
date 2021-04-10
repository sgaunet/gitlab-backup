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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type gitlabResponseSubgroups struct {
	Id      int    `json:"id"`
	Web_url string `json:"web_url"`
}

func getSubgroupsLst(groupID int) (res []gitlabResponseSubgroups, err error) {
	url := fmt.Sprintf("%s/api/v4/groups/%d/subgroups", os.Getenv("GITLAB_URI"), groupID)
	//fmt.Println(url)
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

	var jsonResponse []gitlabResponseSubgroups
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return res, err
	}

	// Loop for every subgroups
	for _, value := range jsonResponse {
		fmt.Printf("Subgroup %d (%s)\n", value.Id, value.Web_url)
		//getProjectsLst(value.Id)
		res = append(res, value)
		recursiveGroups, err := getSubgroupsLst(value.Id)
		if err != nil {
			fmt.Printf("Got an error when trying to get the subgroups of %d (%s)\n", err.Error())
		} else {
			for _, newGroup := range recursiveGroups {
				res = append(res, newGroup)
			}
		}
	}

	return res, err
}
