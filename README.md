[![GitHub release](https://img.shields.io/github/release/sgaunet/gitlab-backup.svg)](https://github.com/sgaunet/gitlab-backup/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/gitlab-backup)](https://goreportcard.com/report/github.com/sgaunet/gitlab-backup)
![GitHub Downloads](https://img.shields.io/github/downloads/sgaunet/gitlab-backup/total)
![Test Coverage](https://raw.githubusercontent.com/wiki/sgaunet/gitlab-backup/coverage-badge.svg)
[![GoDoc](https://godoc.org/github.com/sgaunet/gitlab-backup?status.svg)](https://godoc.org/github.com/sgaunet/gitlab-backup)
[![License](https://img.shields.io/github/license/sgaunet/gitlab-backup.svg)](LICENSE)

# gitlab-backup

This tool can be used to export project or every projects of a gitlab group. It uses the API of gitlab to get an archive of exported project.

Two options to save the exported projects:

* local folder
* s3

There is also the possibility to specify pre/post backup hooks.

# Usage by configuration file


Example: 

```yaml
# debuglevel: "info"
gitlabGroupID: XXXX
gitlabProjectID: YYYY
localpath: "/backup"
gitlabtoken: 
# gitlaburi: https://gitlab.com
# tmpdir: /tmp
hooks:
    prebackup: ""
    postbackup: ""
s3cfg:
  endpoint: "http://localhost:9090"
  bucketName: "ephemeralfiles"
  bucketPath: "test"
  region: "us-east-1"
  accesskey: ""
  secretkey: ""
```

**parameters of the configuration file can be override by environment variable**

Launch the program: `gitlab-backup -c configuration.yaml`

# Usage by environment variable

```
  AWS_ACCESS_KEY_ID string
  AWS_SECRET_ACCESS_KEY string
  GITLABGROUPID int
         (default "0")
  GITLABPROJECTID int
         (default "0")
  GITLAB_TOKEN string
  GITLAB_URI string
         (default "https://gitlab.com")
  LOCALPATH string
         (default "")
  POSTBACKUP string
         (default "")
  PREBACKUP string
         (default "")
  S3BUCKETNAME string
         (default "")
  S3BUCKETPATH string
         (default "")
  S3ENDPOINT string
         (default "")
  S3REGION string
         (default "")
  TMPDIR string
         (default "/tmp")
```

# Extended project

[Another project can be used to encrypt archives of exported project and send them to s3. It's gitlab-backup2s3](https://github.com/sgaunet/gitlab-backup2s3) which is using two softwares:

* gitlab-backup (this project)
* [gocrypt](https://github.com/sgaunet/gocrypt)

# Development

This project is using :

* golang
* [task for development](https://taskfile.dev/#/)
* docker
* [docker buildx](https://github.com/docker/buildx)
* docker manifest
* [goreleaser](https://goreleaser.com/)

Use task to compile/create release...

```bash
$ task
task: [default] task -a
task: Available tasks for this project:
* build:            Build the binary
* default:          List tasks
* doc:              Start godoc server
* image:            Build/push the docker image
* release:          Create a release
* snapshot:         Create a snapshot release
* update-crt:       Update the crt file
```