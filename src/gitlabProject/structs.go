package gitlabProject

type GitlabProject struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type respGitlabExport struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	ExportStatus string `json:"export_status"`
}
