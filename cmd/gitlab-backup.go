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

	"github.com/sgaunet/gitlab-backup/pkg/config"
	"github.com/sgaunet/gitlab-backup/pkg/gitlab"
	"github.com/sgaunet/gitlab-backup/pkg/storage"
	"github.com/sgaunet/gitlab-backup/pkg/storage/localstorage"
	"github.com/sgaunet/gitlab-backup/pkg/storage/s3storage"
	"github.com/sgaunet/ratelimit"
)

var version string = "development"

func printVersion() {
	fmt.Println(version)
}

func main() {
	var wg sync.WaitGroup
	var cfg *config.Config
	var cfgFile string
	var vOption bool
	var printCfg bool
	var storage storage.Storage
	var err error
	paralellTreatment := 2

	// Parameters treatment
	flag.StringVar(&cfgFile, "c", "", "configuration file")
	flag.BoolVar(&printCfg, "cfg", false, "print configuration")
	flag.BoolVar(&vOption, "v", false, "Get version")
	flag.Parse()

	if len(cfgFile) > 0 {
		cfg, err = config.NewConfigFromFile(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		cfg, err = config.NewConfigFromEnv()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if printCfg {
		printConfiguration(cfg)
		os.Exit(0)
	}

	if vOption {
		printVersion()
		os.Exit(0)
	}

	ctx := context.Background()
	r, _ := ratelimit.New(ctx, 60*time.Second, 1)

	log := initTrace(cfg.DebugLevel)
	log.Debug("", "GitlabGroupID", cfg.GitlabGroupID)
	log.Debug("", "GitlabProjectID", cfg.GitlabProjectID)
	log.Debug("", "parallellTreatment", paralellTreatment)
	log.Debug("", "LocalPath", cfg.LocalPath)
	log.Debug("", "AccessKey", cfg.S3cfg.AccessKey)
	log.Debug("", "SecretKey", cfg.S3cfg.SecretKey)
	log.Debug("", "S3Endpoint", cfg.S3cfg.Endpoint)
	log.Debug("", "BucketPath", cfg.S3cfg.BucketPath)
	log.Debug("", "Region", cfg.S3cfg.Region)

	// Check if dst is a S3 URI
	if cfg.IsS3ConfigValid() {
		storage, err = s3storage.NewS3Storage(cfg.S3cfg.Region, cfg.S3cfg.Endpoint, cfg.S3cfg.BucketName, cfg.S3cfg.BucketPath)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
	} else {
		if len(cfg.LocalPath) == 0 {
			log.Error("No storage defined")
			printConfiguration(cfg)
			os.Exit(1)
		}
		storage = localstorage.NewLocalStorage(cfg.LocalPath)
		if stat, err := os.Stat(cfg.LocalPath); err != nil || !stat.IsDir() {
			log.Error(fmt.Sprintf("%s is not a directory\n", cfg.LocalPath))
			os.Exit(1)
		}
	}

	if cfg.GitlabGroupID != 0 {
		s := gitlab.NewService()
		s.SetLogger(log)
		group, err := s.GetGroup(cfg.GitlabGroupID)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
		projects, err := s.GetEveryProjectsOfGroup(group.Id)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		} else {
			cpt := 0
			for project := range projects {
				if projects[project].Archived {
					log.Warn("Project", projects[project].Name, "is archived, skip it")
				} else {
					if cpt == paralellTreatment {
						wg.Wait()
						cpt = 0
					}
					wg.Add(1)
					r.WaitIfLimitReached()
					// go projects[project].SaveProject(&wg, storage, cfg.TmpDir)
					go s.SaveProject(&wg, storage, projects[project].Id, cfg.TmpDir)
					cpt++
				}
			}
		}
	}
	if cfg.GitlabProjectID != 0 {
		s := gitlab.NewService()
		s.SetLogger(log)

		project, err := s.GetProject(cfg.GitlabProjectID)
		if project.Archived {
			log.Warn("Project", project.Name, "is archived, skip it")
		} else {
			if err != nil {
				log.Error(err.Error())
			}
			wg.Add(1)
			// go project.SaveProject(&wg, storage, cfg.TmpDir)
			s.SaveProject(&wg, storage, project.Id, cfg.TmpDir)
		}
	}
	wg.Wait()
}
