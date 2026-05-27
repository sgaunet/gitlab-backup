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
* Native [age](https://age-encryption.org) encryption of archives (optional, recipient public keys)
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
# exportTimeoutMins: 10  # Export timeout in minutes (default: 1440, increase for large projects)
# importTimeoutMins: 60  # Import timeout in minutes for gitlab-restore (default: 60, max: 1440)
hooks:
    prebackup: ""
    postbackup: ""
# Optional: encrypt archives with age before upload. Recipients are PUBLIC keys.
# The matching private identity must stay offline and is only used for restore.
# age:
#   recipients:
#     - age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
#   recipientsFile: /etc/age/recipients.txt   # alternative to inline list
#   armor: false                              # true → PEM/ASCII output
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
         (default "1440")
  IMPORT_TIMEOUT_MIN int
         (default "60")
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
  AGE_RECIPIENTS string
         (comma-separated public keys: age1..., ssh-ed25519 ..., ssh-rsa ...)
  AGE_RECIPIENTS_FILE string
         (path to a file containing one recipient per line, # for comments)
  AGE_ARMOR bool
         (default "false"; true → ASCII-armored .age output)
```

# Archive encryption with age

`gitlab-backup` can encrypt every produced archive in place using the [age](https://age-encryption.org)
file encryption format. Encryption happens **after** the postbackup hook runs and **before** the
archive is uploaded to S3 or written to local storage, so the file at rest is always encrypted.

age is asymmetric: the *recipient* (public key) lives wherever the backup runs, and the *identity*
(private key) stays offline. A compromised backup runner can produce new encrypted archives but
cannot read past ones.

## Generating a key pair

Use the `age-keygen` CLI from the [age release](https://github.com/FiloSottile/age/releases):

```bash
age-keygen -o backup-key.txt
# backup-key.txt contains a line like:
#   # public key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
# followed by the private identity AGE-SECRET-KEY-1...
```

Store `backup-key.txt` somewhere offline (password manager, vault). Copy the
`# public key:` line into the configuration of the backup runner.

## Enabling encryption

Either YAML:

```yaml
age:
  recipients:
    - age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
  armor: false
```

Or env vars (multiple recipients comma-separated, no spaces):

```bash
export AGE_RECIPIENTS="age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p,age1xxxxx..."
# or, for a mounted recipients file:
export AGE_RECIPIENTS_FILE=/etc/age/recipients.txt
```

SSH public keys (`ssh-ed25519`, `ssh-rsa` ≥ 2048 bits) also work as recipients.

Multiple recipients are encrypted to as a list — *any one* of them can decrypt. Useful for a
primary key + an offline recovery key.

## Restoring an encrypted archive

The encrypted archive still has the same `.tar.gz` filename — only the bytes change.
Decrypt locally with your offline identity:

```bash
# binary (default) output:
age -d -i backup-key.txt -o myproject-42.tar.gz s3-downloaded-archive
tar tzf myproject-42.tar.gz | head

# ASCII-armored output (AGE_ARMOR=true) works the same way:
age -d -i backup-key.txt -o myproject-42.tar.gz armored-archive
```

`gitlab-restore` does **not** decrypt automatically — decrypt first, then pass the plaintext
archive to `gitlab-restore --archive`.

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
exportTimeoutMins: 1440
importTimeoutMins: 60   # max wait for the GitLab import to finish; bump for large projects (max 1440)

# For S3 restores
s3cfg:
  endpoint: "https://s3.amazonaws.com"
  bucketName: "backup-bucket"
  bucketPath: "gitlab-backups"
  region: "us-east-1"
```

> Tip: if the import takes longer than `importTimeoutMins`, `gitlab-restore` exits with a clear timeout
> message pointing at the project URL. The import may still finish on the GitLab side — check the web UI
> before retrying. Override via `IMPORT_TIMEOUT_MIN=120` (env) or `importTimeoutMins: 120` (YAML).

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

[gitlab-backup2s3](https://github.com/sgaunet/gitlab-backup2s3) wraps this tool in a container
image with extra tooling and a Helm chart for running scheduled backups in Kubernetes. It bundles:

* `gitlab-backup` (this project) — native age encryption support
* [age](https://github.com/FiloSottile/age) — recommended encryption tool
* [gocrypt](https://github.com/sgaunet/gocrypt) — kept for backward compatibility with existing setups

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