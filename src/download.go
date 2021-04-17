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
	"fmt"
	"io"
	"net/http"
	"os"
)

func downloadProject(project respGitlabExport, dirToSaveFile string) error {
	tmpFile := dirToSaveFile + string(os.PathSeparator) + project.Name + ".tar.gz.tmp"
	finalFile := dirToSaveFile + string(os.PathSeparator) + project.Name + ".tar.gz"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v4/projects/%d/export/download", os.Getenv("GITLAB_URI"), project.Id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", os.Getenv("GITLAB_TOKEN"))
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	// fmt.Println("Taille: ", resp.ContentLength)
	// fmt.Printf("Downloading %s\n", project.Name)

	if _, err = io.Copy(out, resp.Body); err != nil {
		out.Close()
		return err
	}
	out.Close()

	if err = os.Rename(tmpFile, finalFile); err != nil {
		return err
	}
	return nil
}
