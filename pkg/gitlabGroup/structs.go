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

package gitlabGroup

import "github.com/sgaunet/gitlab-backup/pkg/gitlabProject"

type GitlabGroup interface {
	GetProjectsLst() (res []gitlabProject.GitlabProject, err error)
	GetEveryProjectsOfGroup() (res []gitlabProject.GitlabProject, err error)
	GetSubgroupsLst() (res []GitlabGroup, err error)
	GetID() int
}

type gitlabGroup struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type respGitlabProject struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
