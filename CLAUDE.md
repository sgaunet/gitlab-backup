# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains two complementary Go CLI tools for GitLab project backup and restore operations. The project follows clean architecture principles with clear separation between API client, business logic, and storage layers.

### gitlab-backup

**Key Features:**
- Exports GitLab projects using the native GitLab export API
- Supports both local filesystem and AWS S3 storage
- Implements rate limiting for all GitLab API endpoints
- Concurrent group exports with proper error handling

### gitlab-restore

**Key Features:**
- Restores GitLab projects from archives created by gitlab-backup
- Validates target project is empty before restoring (with optional override)
- Restores complete project using GitLab's native Import/Export API (includes repository, wiki, issues, merge requests, labels, and all project data)
- Progress reporting for each restore phase (validation, download, extraction, import, cleanup)
- Graceful interruption handling (Ctrl+C)
- Supports both local and S3-stored archives

## Development Commands

**Build & Development** (using [Task](https://taskfile.dev/)):
- `task build` - Build both binaries (gitlab-backup and gitlab-restore)
- `task linter` - Run golangci-lint with project configuration
- `task doc` - Start godoc server for documentation
- `task image` - Build Docker image
- `task snapshot` - Create snapshot release with goreleaser (both binaries)
- `task release` - Create production release (both binaries)

**Testing**:
- `go test ./...` - Run all tests
- Tests use testify framework and testcontainers for integration testing

**Prerequisites**: Go 1.23.0+, Task, Docker (for images)

## Architecture

**Clean Architecture Pattern**:
- `cmd/` - CLI entry points and argument parsing
  - `cmd/gitlab-backup/` - Backup CLI tool
  - `cmd/gitlab-restore/` - Restore CLI tool
- `pkg/app/` - Application orchestration and business logic
  - `pkg/app/restore/` - Restore workflow orchestration, validation, and progress reporting
- `pkg/config/` - Configuration management (YAML/ENV)
  - `restore_config.go` - Restore-specific configuration and validation
- `pkg/gitlab/` - GitLab API client with rate limiting
  - `client_interface.go` - Interface definitions and wrappers for GitLab API services
  - `gitlab.go` - Service initialization and rate limiter configuration
  - `project.go` - Project export orchestration
  - `restore.go` - Project import via GitLab's native Import/Export API
- `pkg/storage/` - Storage abstraction (local/S3 implementations)
  - `archive.go` - Archive validation and extraction with path traversal protection (backward compatible with old archive formats)
- `pkg/hooks/` - Pre/post backup hook execution

**Key Interfaces**:
- `Storage` interface with `localstorage` and `s3storage` implementations
  - Backup: `SaveFile()` - Store archives to local or S3
  - Restore: `Get()` - Retrieve archives from local or S3
- `GitLabClient` interface with service-specific interfaces:
  - `GroupsService` - Group operations
  - `ProjectsService` - Project metadata
  - `ProjectImportExportService` - Project export/import operations
    - Export: `ExportProject()`, `ExportStatus()`
    - Import: `ImportFromFile()`, `ImportStatus()`
  - `LabelsService` - Used for restore validation (checking project emptiness)
  - `IssuesService` - Used for restore validation (checking project emptiness)
  - `CommitsService` - Used for restore validation (checking project emptiness)
- Interface-based design enables easy testing and extensibility

**Rate Limiting**:
- Download API: 5 requests/minute (GitLab repository files API limit)
- Export API: 6 requests/minute (GitLab project import/export API limit)
- Import API: 6 requests/minute (GitLab project import/export API limit)

**Concurrency**: Uses `golang.org/x/sync/errgroup` for concurrent project exports with GitLab API rate limiting

**Archive Strategy**: Creates standard tar.gz archives using GitLab's native export API. The archive contains the complete project including repository, wiki, issues, merge requests, labels, and all other project data.

## Configuration

Configuration supports both YAML files and environment variables with override capability. The config package handles validation for different storage backends and restore options.

**Restore Configuration** (`restore_config.go`):
- `restoreSource` (RESTORE_SOURCE): Archive path (local or s3://bucket/key)
- `restoreTargetNS` (RESTORE_TARGET_NS): Target GitLab namespace/group path
- `restoreTargetPath` (RESTORE_TARGET_PATH): Target GitLab project name
- `restoreOverwrite` (RESTORE_OVERWRITE): Skip emptiness validation (default: false)

The restore configuration extends the base Config struct and includes validation for:
- S3 path parsing (s3://bucket/key format)
- Project path format validation (alphanumeric, underscores, dots, hyphens)
- Path traversal prevention
- S3 configuration requirements when using S3 sources

## Restore Architecture

**5-Phase Restore Workflow** (`pkg/app/restore/restore.go`):

1. **Phase 1: Validation** - Verify target project is empty
   - Check for commits (via CommitsService)
   - Check for issues (via IssuesService)
   - Check for labels (via LabelsService)
   - Skip if `--overwrite` flag set

2. **Phase 2: Download** - Download archive from S3 (if S3 source)
   - Parse S3 path (s3://bucket/key)
   - Download to temporary file
   - Track download progress

3. **Phase 3: Extraction** - Extract archive contents
   - Validate tar.gz format
   - Extract to temporary directory with path traversal protection
   - Verify presence of required files (project.tar.gz)
   - Backward compatible: silently ignores old labels.json/issues.json files

4. **Phase 4: Import** - Import complete GitLab project
   - Upload project export via ImportFromFile API
   - Poll ImportStatus with 5-second interval, 10-minute timeout
   - Wait for completion or failure
   - GitLab's native import handles repository, wiki, issues, merge requests, labels, and all project data

5. **Phase 5: Cleanup** - Remove temporary files
   - Delete extraction directory
   - Delete downloaded S3 archive (if applicable)
   - Always runs (defer)

**Error Handling**:
- **Fatal errors** (phases 1-4): Stop restore, return error
- GitLab's native import is atomic - either succeeds completely or fails

**Validation** (`pkg/app/restore/validator.go`):
- `ValidateProjectEmpty()` checks three conditions:
  - No commits in repository
  - No issues in project
  - No labels in project
- Returns `EmptinessChecks` struct with counts for each category
- Prevents accidental data overwrite

**Progress Reporting** (`pkg/app/restore/progress.go`):
- Console logger implementation with phase-based progress
- Reports: phase start, phase complete, phase fail, phase skip
- Tracks metrics: labels/issues restored, notes created, duration

## Testing Strategy

- Unit tests for each package using testify
- Integration tests with testcontainers-go
- Both white-box and black-box testing approaches
- GitHub Actions workflow maintains coverage badges

## Docker

Multi-stage builds create minimal scratch-based images with non-root user. Images use GitHub Container Registry (ghcr.io).

## Active Technologies
- Go 1.24.0+

## Recent Changes
- **002-remove-metadata-export** (2026-02): Removed separate labels/issues export/restore
  - Simplified to use GitLab's native export/import exclusively
  - Removed 800+ lines of metadata export/restore code
  - Changed from 7-phase to 5-phase restore workflow
  - Removed composite archive creation
  - Backward compatible: old archives with labels.json/issues.json still work
  - Faster backups and restores (fewer API calls)
  - More reliable (GitLab handles all data relationships internally)

- **001-gitlab-restore** (2026-02): Added gitlab-restore CLI tool
  - Complete restore workflow with 5-phase orchestration
  - Project emptiness validation
  - GitLab Import/Export API integration
  - S3 archive support
  - Progress reporting and graceful interruption handling
  - Multi-platform binary releases via GoReleaser
  - Extended GitLab client interfaces for import/restore operations
  - Archive extraction with path traversal protection
  - Comprehensive test coverage with testify and testcontainers
