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
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sgaunet/gitlab-backup/gitlabProject"
)

func main() {
	var gid int        // Gitlab Group ID parent to backup
	var pid int        // Gitlab Project ID to backup
	var dirpath string // path to save archives
	var wg sync.WaitGroup

	// Parameters treatment
	flag.IntVar(&gid, "gid", 0, "Gitlab Group ID parent to backup")
	flag.IntVar(&pid, "pid", 0, "Gitlab Project ID to backup")
	flag.StringVar(&dirpath, "o", ".", "Path to save archives")
	flag.Parse()

	if stat, err := os.Stat(dirpath); err != nil || !stat.IsDir() {
		fmt.Printf("%s is not a directory\n", dirpath)
		os.Exit(1)
	}

	if gid == 0 && pid == 0 {
		fmt.Println("Parameter gid or pid is mandatory")
		os.Exit(1)
	}

	if len(os.Getenv("GITLAB_TOKEN")) == 0 {
		fmt.Println("Set GITLAN_TOKEN environment variable")
		os.Exit(1)
	}

	if len(os.Getenv("GITLAB_URI")) == 0 {
		os.Setenv("GITLAB_URI", "https://gitlab.com")
	}

	if gid != 0 {
		projects, err := getEveryProjectsOfGroup(gid)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		} else {
			for _, project := range projects {
				wg.Add(1)
				saveProjectOnDisk(project, dirpath, &wg)
			}
		}
	}
	if pid != 0 {
		project, err := gitlabProject.New(pid)
		if err != nil {
			fmt.Println(err.Error())
		}
		wg.Add(1)
		go saveProjectOnDisk(project, dirpath, &wg)
	}
	wg.Wait()
}

func saveProjectOnDisk(project gitlabProject.GitlabProject, dirpath string, wg *sync.WaitGroup) (err error) {
	defer wg.Done()
	statuscode := 0
	// fmt.Println("\tAsk export for project", project.Name)
	for statuscode != 202 {
		fmt.Printf("%s : Ask gitlab to export a backup\n", project.Name)
		statuscode, err = askExportForProject(project.Id)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		time.Sleep(20 * time.Second)
	}
	fmt.Printf("%s : Gitlab is creating the archive\n", project.Name)
	gitlabExport, err := waitForExport(project.Id)
	if err != nil {
		fmt.Printf("%s: Export failed, reason: %s\n", project.Name, err.Error())
		return errors.New("Failed ...")
	}
	fmt.Printf("%s : Gitlab has created the archive, download is beginning\n", project.Name)
	downloadProject(gitlabExport, dirpath)
	fmt.Printf("%s : Succesfully exported\n", project.Name)
	return nil
}

func getEveryProjectsOfGroup(gid int) (res []gitlabProject.GitlabProject, err error) {
	subgroups, err := getSubgroupsLst(gid)
	if err != nil {
		fmt.Printf("Got error when listing subgroups of %d (%s)\n", gid, err.Error())
		os.Exit(1)
	}
	for _, group := range subgroups {
		//fmt.Println("ID =>", v.Id)
		projects, err := getProjectsLst(group.Id)
		if err != nil {
			fmt.Printf("Got error when listing projects of %d (%s)\n", group.Id, err.Error())
		} else {
			for _, project := range projects {
				fmt.Println("project ID:", project.Name)
				res = append(res, project)
			}
		}
	}

	projects, err := getProjectsLst(gid)
	if err != nil {
		fmt.Printf("Got error when listing projects of %d (%s)\n", gid, err.Error())
	} else {
		for _, project := range projects {
			fmt.Println("project:", project.Name)
			res = append(res, project)
		}
	}
	return res, err
}
