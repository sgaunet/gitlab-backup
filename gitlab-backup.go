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
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sgaunet/gitlab-backup/gitlabGroup"
	"github.com/sgaunet/gitlab-backup/gitlabProject"
	"github.com/sgaunet/ratelimit"
	log "github.com/sirupsen/logrus"
)

func initTrace(debugLevel string) {
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})
	// log.SetFormatter(&log.TextFormatter{
	// 	DisableColors: true,
	// 	FullTimestamp: true,
	// })

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	switch debugLevel {
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.DebugLevel)
	}
}

var version string = "development"

func printVersion() {
	fmt.Println(version)
}

func main() {
	var gid int        // Gitlab Group ID parent to backup
	var pid int        // Gitlab Project ID to backup
	var dirpath string // path to save archives
	var debugLevel string
	var wg sync.WaitGroup
	var paralellTreatment int
	var vOption bool

	// Parameters treatment
	flag.IntVar(&gid, "gid", 0, "Gitlab Group ID parent to backup")
	flag.IntVar(&pid, "pid", 0, "Gitlab Project ID to backup")
	flag.IntVar(&paralellTreatment, "p", 2, "Number of projects to treat in parallel")
	flag.StringVar(&dirpath, "o", ".", "Path to save archives")
	flag.StringVar(&debugLevel, "d", "info", "debuglevel : debug/info/warn/error")
	flag.BoolVar(&vOption, "v", false, "Get version")
	flag.Parse()

	if vOption {
		printVersion()
		os.Exit(0)
	}
	ctx := context.Background()
	r, _ := ratelimit.New(ctx, 60*time.Second, 1)

	initTrace(debugLevel)
	log.Debugf("gid=%d\n", gid)
	log.Debugf("pid=%d\n", pid)
	log.Debugf("parallellTreatment=%d\n", paralellTreatment)
	log.Debugf("dirpath=%s\n", dirpath)

	if stat, err := os.Stat(dirpath); err != nil || !stat.IsDir() {
		log.Errorf("%s is not a directory\n", dirpath)
		os.Exit(1)
	}

	if gid != 0 && paralellTreatment <= 0 {
		log.Errorln("Value incorrect for option -p (should be greater than 0)")
		os.Exit(1)
	}

	if gid == 0 && pid == 0 {
		log.Errorln("Parameter gid or pid is mandatory")
		os.Exit(1)
	}

	if len(os.Getenv("GITLAB_TOKEN")) == 0 {
		log.Errorln("Set GITLAB_TOKEN environment variable")
		os.Exit(1)
	}

	if len(os.Getenv("GITLAB_URI")) == 0 {
		os.Setenv("GITLAB_URI", "https://gitlab.com")
	}

	log.Debugf("GITLAB_TOKEN=%s\n", os.Getenv("GITLAB_TOKEN"))
	log.Debugf("GITLAB_URI=%s\n", os.Getenv("GITLAB_URI"))

	if gid != 0 {
		group, err := gitlabGroup.New(gid)
		if err != nil {
			log.Errorln(err.Error())
			os.Exit(1)
		}
		projects, err := group.GetEveryProjectsOfGroup()
		if err != nil {
			log.Errorln(err.Error())
			os.Exit(1)
		} else {
			cpt := 0
			for project := range projects {
				if projects[project].IsArchived() {
					log.Warningln("Project", projects[project].GetName(), "is archived, skip it")
				} else {
					if cpt == paralellTreatment {
						wg.Wait()
						cpt = 0
					}
					wg.Add(1)
					r.WaitIfLimitReached()
					go projects[project].SaveProjectOnDisk(dirpath, &wg)
					cpt++
				}
			}
		}
	}
	if pid != 0 {
		project, err := gitlabProject.New(pid)
		if project.IsArchived() {
			log.Warningln("Project", project.GetName(), "is archived, skip it")
		} else {
			if err != nil {
				log.Errorln(err.Error())
			}
			wg.Add(1)
			go project.SaveProjectOnDisk(dirpath, &wg)
		}
	}
	wg.Wait()
}
