package gitlabProject

import "sync"

type GitlabProject interface {
	SaveProjectOnDisk(dirpath string, wg *sync.WaitGroup) (err error)
	GetName() string
	GetID() int
}

type gitlabProject struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type respGitlabExport struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	ExportStatus string `json:"export_status"`
}
