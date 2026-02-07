[![GitHub release](https://img.shields.io/github/release/sgaunet/gitlab-backup.svg)](https://github.com/sgaunet/gitlab-backup/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/gitlab-backup)](https://goreportcard.com/report/github.com/sgaunet/gitlab-backup)
![GitHub Downloads](https://img.shields.io/github/downloads/sgaunet/gitlab-backup/total)
![Test Coverage](https://raw.githubusercontent.com/wiki/sgaunet/gitlab-backup/coverage-badge.svg)
[![GoDoc](https://godoc.org/github.com/sgaunet/gitlab-backup?status.svg)](https://godoc.org/github.com/sgaunet/gitlab-backup)
[![Linter](https://github.com/sgaunet/gitlab-backup/workflows/linter/badge.svg)](https://github.com/sgaunet/gitlab-backup/actions/workflows/linter.yml)
[![Release](https://github.com/sgaunet/gitlab-backup/workflows/release/badge.svg)](https://github.com/sgaunet/gitlab-backup/actions/workflows/release.yml)
[![Snapshot](https://github.com/sgaunet/gitlab-backup/workflows/snapshot/badge.svg)](https://github.com/sgaunet/gitlab-backup/actions/workflows/snapshot.yml)
[![License](https://img.shields.io/github/license/sgaunet/gitlab-backup.svg)](LICENSE)

# gitlab-backup

This tool can be used to export project or every projects of a gitlab group. It uses the API of gitlab to get an archive of exported project.

**Features:**
* Export GitLab projects or entire groups using GitLab's native export API
* Two storage options: local folder or S3
* Pre/post backup hooks support
* Configurable rate limiting for GitLab API
* Concurrent project exports for groups

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
# exportTimeoutMins: 10  # Export timeout in minutes (default: 10, increase for large projects)
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

## Archive Structure

The tool creates a standard tar.gz archive using GitLab's native project export API. The archive contains the complete project including repository, wiki, issues, merge requests, labels, and all other project data.

The final archive is named: `{projectName}-{projectID}.tar.gz`

**parameters of the configuration file can be override by environment variable**

Launch the program: `gitlab-backup -c configuration.yaml`

# Usage by CLI Flags

`gitlab-backup` now supports rich command-line arguments that can override configuration file and environment variable settings. This makes the tool easier to use for manual invocations and quick backups.

## Configuration Precedence

Settings are applied in this order (highest priority first):

1. **CLI flags** (`--project-id 123`)
2. **Configuration file** (`gitlabProjectID: 123`)
3. **Environment variables** (`GITLABPROJECTID=123`)

## Available Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--config`, `-c` | Path to YAML configuration file | (optional) |
| `--group-id` | GitLab group ID to backup | 0 |
| `--project-id` | GitLab project ID to backup | 0 |
| `--output` | Output directory for local storage | "" |
| `--timeout` | Export timeout in minutes | 10 |
| `--tmpdir` | Temporary directory | /tmp |
| `--gitlab-url` | GitLab API endpoint | https://gitlab.com |
| `--version`, `-v` | Show version and exit | |
| `--help`, `-h` | Show help message | |
| `--cfg` | Print configuration and exit | |

## CLI Examples

### Minimal CLI-only Usage

```bash
export GITLAB_TOKEN=glpat-xxxxx
gitlab-backup --project-id 123 --output /backup
```

### Backup Entire Group

```bash
export GITLAB_TOKEN=glpat-xxxxx
gitlab-backup --group-id 456 --output /backup/groups
```

### Override Config File Values

```bash
# Use config for most settings, but increase timeout
gitlab-backup --config production.yaml --timeout 30
```

### Backup to Custom GitLab Instance

```bash
gitlab-backup \
  --project-id 123 \
  --output /backup \
  --gitlab-url https://gitlab.mycompany.com \
  --timeout 60
```

## Required Settings

These settings must be provided via one of the configuration methods:

1. **GitLab Token**: Set via `GITLAB_TOKEN` env var (recommended) or `gitlabtoken` in config file
2. **Project or Group ID**: Set via `--project-id` or `--group-id` (choose one)
3. **Storage**: Set via `--output` for local storage, or `s3cfg` section in config file

**Note**: S3 storage requires a config file with the `s3cfg` section, as it involves multiple settings (bucket, region, credentials).

# Usage by environment variable

```
  AWS_ACCESS_KEY_ID string
  AWS_SECRET_ACCESS_KEY string
  EXPORT_TIMEOUT_MIN int
         (default "10")
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

# gitlab-restore

The `gitlab-restore` tool restores GitLab projects from archives created by `gitlab-backup`. It validates that target projects are empty before proceeding and restores the repository, labels, and issues.

**Features:**
* Restore GitLab projects from local or S3-stored archives
* Validate target project is empty before restoring
* Restore complete project using GitLab's native Import/Export API (includes repository, wiki, issues, merge requests, labels, and all project data)
* Progress reporting for each restore phase
* Graceful interruption handling (Ctrl+C)
* Multi-platform support (Linux, macOS, Windows)

## Restore Usage

### Basic Restore from Local Archive

```bash
gitlab-restore \
  --config config.yml \
  --archive /path/to/backup.tar.gz \
  --namespace mygroup \
  --project restored-project
```

### Restore from S3 Archive

```bash
gitlab-restore \
  --config config.yml \
  --archive s3://bucket/path/to/backup.tar.gz \
  --namespace mygroup \
  --project restored-project
```

### Overwrite Existing Project

**⚠️ Use with caution:** Skip emptiness validation

```bash
gitlab-restore \
  --config config.yml \
  --archive /path/to/backup.tar.gz \
  --namespace mygroup \
  --project existing-project \
  --overwrite
```

## Restore Configuration File

The restore tool uses the same configuration file as `gitlab-backup`:

```yaml
gitlabtoken: "your-gitlab-token"
gitlaburi: "https://gitlab.com"
tmpdir: "/tmp"
exportTimeoutMins: 10

# For S3 restores
s3cfg:
  endpoint: "https://s3.amazonaws.com"
  bucketName: "backup-bucket"
  bucketPath: "gitlab-backups"
  region: "us-east-1"
```

**Configuration can be overridden by environment variables**

### Restore-Specific Environment Variables

```
RESTORE_SOURCE string
       Archive path (local or s3://bucket/key)
RESTORE_TARGET_NS string
       Target GitLab namespace/group
RESTORE_TARGET_PATH string
       Target GitLab project name
RESTORE_OVERWRITE bool
       Skip emptiness validation (default "false")
```

## Restore Process Phases

The restore operation proceeds through these phases:

1. **Validation** - Verify target project is empty (skip with `--overwrite`)
2. **Download** - Download archive from S3 (if S3 source)
3. **Extraction** - Extract archive contents to temporary directory
4. **Import** - Import complete project via GitLab's Import/Export API (includes repository, wiki, issues, merge requests, labels, and all project data)
5. **Cleanup** - Remove temporary files

## Restore Requirements

* Target GitLab project must exist (create it first via GitLab UI or API)
* User must have **Maintainer** or **Owner** permissions on target project
* For S3 restores: AWS credentials with read permissions
* Archive must be created by `gitlab-backup` (tar.gz format)

## Installation

Download both `gitlab-backup` and `gitlab-restore` from [github releases](https://github.com/sgaunet/gitlab-backup/releases).

### Brew

```bash
brew tap sgaunet/homebrew-tools
brew install sgaunet/tools/gitlab-backup
brew install sgaunet/tools/gitlab-restore
```

# Extended project

[Another project can be used to encrypt archives of exported project and send them to s3. It's gitlab-backup2s3](https://github.com/sgaunet/gitlab-backup2s3) which is using two softwares:

* gitlab-backup (this project)
* [gocrypt](https://github.com/sgaunet/gocrypt)

# Installation

## Release

Download the latest release from [github](https://github.com/sgaunet/gitlab-backup/releases) and install it.

## Brew

```bash
brew tap sgaunet/homebrew-tools
brew install sgaunet/tools/gitlab-backup
```

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