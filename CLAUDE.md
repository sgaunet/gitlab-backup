# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Repository Overview

Two complementary Go CLI tools for GitLab backup and restore operations. Uses GitLab's native export/import API with clean architecture, interface-driven design, and AWS S3 support. Built with Go 1.24.0+ using Task for build automation.

**gitlab-backup**: Exports GitLab projects with rate limiting, concurrent group exports, and local/S3 storage.

**gitlab-restore**: Restores from archives with 5-phase workflow (validation, download, extraction, import, cleanup). Validates target emptiness before restore.

## Architecture

**Clean Architecture with Interface-Driven Design**:
- `cmd/` - CLI entry points (gitlab-backup, gitlab-restore)
- `pkg/app/` - Application orchestration and business logic
- `pkg/config/` - Configuration management (YAML/ENV)
- `pkg/constants/` - Centralized configuration constants (rate limits, timeouts, validation rules)
- `pkg/gitlab/` - GitLab API client with rate limiting and service interfaces
- `pkg/storage/` - Storage abstraction (local/S3 implementations)
- `pkg/hooks/` - Pre/post backup hook execution

**Key Patterns**:
- Interface-based dependency injection (`GitLabClient`, `Storage`)
- Sentinel errors with wrapping (`pkg/gitlab/error.go`)
- Rate limiting per API endpoint (golang.org/x/time/rate)
- Structured concurrency (golang.org/x/sync/errgroup)
- White-box, black-box, and integration testing strategies

See [docs/architecture.md](docs/architecture.md) for detailed design, component overview, and restore workflow.

## Development Commands

**Build & Development** (using [Task](https://taskfile.dev/)):
```bash
task build        # Build both binaries
task test         # Run tests with race detection
task lint         # Run golangci-lint
task doc          # Start godoc server
task snapshot     # Create snapshot release
task release      # Create production release
```

**Testing**:
- `go test -count=2 -race ./...` - Run all tests
- Uses testify framework and testcontainers for integration tests

**Prerequisites**: Go 1.24.0+, Task, Docker (for images)

## Code Quality Standards

**Linter configured** (do not duplicate rules):
- **golangci-lint**: See [.golangci.yml](.golangci.yml) for complete rule set (all linters enabled, 11 disabled)

**Key conventions**:
- Test files: `_test.go` suffix with white-box/black-box approaches
- Errors: Sentinel errors (exported vars) with `fmt.Errorf("%w")` wrapping
- Mocks: Generated via `go:generate` directives using moq

## File Locations

- **Source**: `pkg/` (app, config, gitlab, storage, hooks packages)
- **Tests**: `tests/integration/` + unit tests co-located with source
- **Docs**: `docs/` (architecture, workflows, patterns)
- **Config**: `resources/` (example configs), `pkg/config/testdata/`
- **Binaries**: `cmd/gitlab-backup/`, `cmd/gitlab-restore/`

## Configuration

Supports YAML files and environment variables with override capability. Config package handles validation for storage backends and restore options.

**Key restore options**: `restoreSource` (local/S3), `restoreTargetNS`, `restoreTargetPath`, `restoreOverwrite` (skip emptiness check).

See [docs/architecture.md](docs/architecture.md) for detailed configuration reference.

## Constants

**Centralized Configuration**: All hard-coded values, timeouts, and limits are in `pkg/constants/`:
- `gitlab.go` - GitLab API constants (rate limits, timeouts, endpoints)
- `storage.go` - Storage constants (buffer sizes, file permissions)
- `validation.go` - Validation constraints (AWS limits, config boundaries)
- `output.go` - CLI output formatting constants

**Tuning**: Most constants are based on external API limits (GitLab, AWS). Before modifying, check the godoc comments for rationale and external references.

**Key Constants**:
- Rate limits: Based on GitLab documented API limits (5-6 req/min)
- Timeouts: Default 10 minutes for export/import operations
- Buffer sizes: 32KB for file I/O operations

## Documentation

- [docs/architecture.md](docs/architecture.md): System design, components, restore workflow, rate limits, configuration
- [docs/workflows.md](docs/workflows.md): Development processes, testing strategy, git workflow, CI/CD
- [docs/patterns.md](docs/patterns.md): Code patterns, error handling, testing approaches, concurrency

## Docker

Multi-stage builds create minimal scratch-based images with non-root user. Images use GitHub Container Registry (ghcr.io).

## Active Technologies
- Go 1.24.0+ (toolchain: go1.24.3)
- GitLab API client (gitlab.com/gitlab-org/api/client-go)
- AWS SDK v2 (S3 storage)
- testify, testcontainers-go (testing)

## Recent Changes
- **002-remove-metadata-export** (2026-02): Simplified to GitLab native export/import, removed 800+ lines, faster and more reliable
- **001-gitlab-restore** (2026-02): Added restore CLI with 5-phase workflow, emptiness validation, S3 support, progress reporting
