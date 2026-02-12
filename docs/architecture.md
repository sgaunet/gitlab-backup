# Architecture

## System Overview

gitlab-backup is a monorepo containing two complementary CLI tools that leverage GitLab's native export/import API for backup and restore operations. The architecture follows clean architecture principles with strict separation of concerns through interface-driven design.

## Components

### Core Layers

**cmd/** - CLI Entry Points
- `cmd/gitlab-backup/` - Backup tool entry point and argument parsing
- `cmd/gitlab-restore/` - Restore tool entry point and argument parsing

**pkg/app/** - Application Orchestration
- `pkg/app/` - Backup workflow orchestration
- `pkg/app/restore/` - Restore workflow orchestration, validation, progress reporting

**pkg/gitlab/** - GitLab API Client
- `client_interface.go` - Interface definitions and wrappers for GitLab API services
- `gitlab.go` - Service initialization and rate limiter configuration
- `project.go` - Project export orchestration
- `restore.go` - Project import via GitLab's native Import/Export API
- `error.go` - Sentinel errors and error handling

**pkg/storage/** - Storage Abstraction
- `storage.go` - Storage interface definition
- `localstorage/` - Local filesystem implementation
- `s3storage/` - AWS S3 implementation
- `archive.go` - Archive validation and extraction with path traversal protection

**pkg/config/** - Configuration Management
- `config.go` - Base configuration with YAML/ENV support
- `restore_config.go` - Restore-specific configuration and validation

**pkg/constants/** - Centralized Configuration Constants
Single source of truth for all hard-coded values, timeouts, and limits.
Organized by topic (GitLab, storage, validation, output) with comprehensive
documentation and external API references.

**pkg/hooks/** - Hook Execution
- Pre/post backup hook execution

## Key Interfaces

### GitLabClient Interface
Wraps GitLab API client with service-specific interfaces:
- `GroupsService` - Group operations
- `ProjectsService` - Project metadata
- `ProjectImportExportService` - Export/import operations
  - `ExportProject()`, `ExportStatus()` - Export workflow
  - `ImportFromFile()`, `ImportStatus()` - Import workflow
- `LabelsService` - Restore validation (project emptiness check)
- `IssuesService` - Restore validation (project emptiness check)
- `CommitsService` - Restore validation (project emptiness check)

Location: `pkg/gitlab/client_interface.go:18-74`

### Storage Interface
Provides abstraction for archive storage:
- `SaveFile(ctx, archivePath, destPath)` - Store archives (backup)
- `Get(ctx, sourcePath, destPath)` - Retrieve archives (restore)

Implementations:
- `LocalStorage` (`pkg/storage/localstorage/`) - Filesystem storage
- `S3Storage` (`pkg/storage/s3storage/`) - AWS S3 storage

Location: `pkg/storage/storage.go:8-11`

## Rate Limiting

GitLab API has strict rate limits enforced using `golang.org/x/time/rate` token bucket algorithm:

- **Download API**: 5 requests/minute (repository files API)
- **Export API**: 6 requests/minute (project import/export API)
- **Import API**: 6 requests/minute (project import/export API)

Each API endpoint has a dedicated rate limiter to prevent exceeding GitLab's limits. Implementation: `pkg/gitlab/gitlab.go:25-103`

## Restore Workflow (5 Phases)

**Phase 1: Validation**
- Verify target project is empty via `ValidateProjectEmpty()`
- Checks: no commits, no issues, no labels
- Skipped if `--overwrite` flag set
- Implementation: `pkg/app/restore/validator.go`

**Phase 2: Download** (if S3 source)
- Parse S3 path (s3://bucket/key format)
- Download archive to temporary file
- Track download progress
- Implementation: `pkg/app/restore/restore.go:100-150`

**Phase 3: Extraction**
- Validate tar.gz format
- Extract to temporary directory with path traversal protection
- Verify required files (project.tar.gz)
- Backward compatible with old archive formats
- Implementation: `pkg/storage/archive.go`

**Phase 4: Import**
- Upload project export via `ImportFromFile()` API
- Poll `ImportStatus()` with 5-second interval, 10-minute timeout
- GitLab's native import is atomic (all-or-nothing)
- Implementation: `pkg/gitlab/restore.go`

**Phase 5: Cleanup**
- Delete extraction directory
- Delete downloaded S3 archive (if applicable)
- Always runs (deferred)
- Implementation: `pkg/app/restore/restore.go:200-230`

## Error Handling Strategy

**Sentinel Errors**: Exported error variables for known error types
- `ErrGitlabAPI` - GitLab API errors
- `ErrExportTimeout` - Export timeout
- `ErrImportTimeout` - Import timeout
- `ErrProjectNotEmpty` - Restore validation failure
- `ErrArchiveInvalid` - Archive validation failure

Location: `pkg/gitlab/error.go`, `pkg/app/app.go`, `pkg/storage/archive.go`

**Error Wrapping**: Use `fmt.Errorf("%w", err)` to preserve error chains. Enables `errors.Is()` and `errors.As()` checks.

## Configuration

### Base Configuration (Both Tools)
- `gitlabURL` - GitLab instance URL
- `gitlabToken` - Personal access token
- `groupName` - Group to backup (backup only)
- `projectNames` - Specific projects to backup (backup only)
- `storageType` - "local" or "s3"
- `storageFolder` - Local storage path (if local)
- S3 settings: `s3Region`, `s3Bucket`, `s3Endpoint`, `s3AccessKeyID`, `s3SecretAccessKey`

### Restore Configuration
- `restoreSource` - Archive path (local or s3://bucket/key)
- `restoreTargetNS` - Target GitLab namespace/group path
- `restoreTargetPath` - Target project name
- `restoreOverwrite` - Skip emptiness validation (default: false)

**Configuration Precedence**: CLI flags > Environment variables > YAML file

Implementation: `pkg/config/config.go`, `pkg/config/restore_config.go`

## Concurrency

Uses `golang.org/x/sync/errgroup` for structured concurrency:
- Concurrent project exports with proper error propagation
- Respects rate limits via Wait() on rate limiters
- Graceful error handling - first error cancels remaining goroutines

Implementation: `pkg/app/app.go:111-129`

## Archive Strategy

**Format**: Standard tar.gz archives created by GitLab's native export API

**Contents**: Complete project data including:
- Repository (all branches, tags, commits)
- Wiki
- Issues and issue notes
- Merge requests and MR notes
- Labels
- Milestones
- Project settings
- CI/CD variables (encrypted)

**Backward Compatibility**: Old archives with separate labels.json/issues.json files are still supported (silently ignored during restore).

## Design Decisions

1. **Interface-driven design**: Enables testing with mocks, extensibility, and loose coupling
2. **GitLab native API**: Simplifies implementation, ensures completeness, atomic operations
3. **Rate limiting per endpoint**: Prevents GitLab API throttling, respects different endpoint limits
4. **5-phase restore workflow**: Clear separation of concerns, progress reporting, cleanup guarantees
5. **Sentinel errors**: Type-safe error handling, easy error checks with errors.Is()
6. **No database/ORM**: API-driven architecture, stateless operations
