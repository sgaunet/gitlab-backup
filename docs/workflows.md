# Development Workflows

## Feature Development

Standard Git workflow with feature branches:

1. **Create feature branch** from main
   ```bash
   git checkout -b feature/description
   ```

2. **Implement changes** with tests
   - Write unit tests co-located with code
   - Add integration tests in `tests/integration/` if needed
   - Run `task test` frequently

3. **Run linters** before commit
   ```bash
   task lint
   ```

4. **Commit changes** with conventional commit format
   ```bash
   git commit -m "feat: add feature description"
   git commit -m "fix: resolve bug description"
   git commit -m "chore: update dependencies"
   ```

5. **Push and create PR**
   ```bash
   git push -u origin feature/description
   ```

6. **Wait for CI checks** - All 4 GitHub Actions workflows must pass:
   - Linter workflow (golangci-lint)
   - Coverage workflow (tests + coverage badge update)
   - Snapshot workflow (GoReleaser dry-run)
   - Manual checks if applicable

7. **Merge after approval**

## Code Review Process

- All PRs require review before merge
- Automated checks must pass (linter, tests, coverage)
- Review focus areas:
  - Interface contract adherence
  - Error handling with sentinel errors
  - Rate limiting respect
  - Test coverage for new code
  - No security vulnerabilities (path traversal, injection, etc.)

## Testing Strategy

### Unit Tests
- **Location**: Co-located with source code (e.g., `pkg/gitlab/gitlab_test.go`)
- **Naming**: `*_test.go` suffix
- **Two approaches**:
  - **White-box** (`package gitlab`): Access private functions/vars
  - **Black-box** (`package gitlab_test`): Test public API only

**Example**:
```go
// White-box test
package gitlab
func TestInternalFunction(t *testing.T) { ... }

// Black-box test
package gitlab_test
func TestPublicAPI(t *testing.T) { ... }
```

### Integration Tests
- **Location**: `tests/integration/`
- **Framework**: testcontainers-go for GitLab/MinIO containers
- **Coverage**: End-to-end restore scenarios (local, S3, overwrite)
- **Run**: `task test` or `go test -count=2 -race ./...`

### Mock Generation
- **Tool**: moq (github.com/matryer/moq)
- **Trigger**: `go generate ./...`
- **Directives**: In `pkg/gitlab/client_interface.go:9-16`
- **Output**: `pkg/gitlab/mocks/*.go`

### Coverage Requirements
- Coverage badge maintained by GitHub Actions
- Target: High coverage on business logic (app, gitlab, storage packages)
- Integration tests cover critical restore workflow paths

## Release Process

### Automated via GitHub Actions

**Snapshot Builds** (on push to main):
- Workflow: `.github/workflows/snapshot.yml`
- Trigger: Every push to main branch
- Builds: Both binaries (gitlab-backup, gitlab-restore)
- Output: Snapshot artifacts (not published)

**Production Releases** (on tags):
- Workflow: `.github/workflows/release.yml`
- Trigger: Push tag matching `v*` (e.g., `v1.2.3`)
- Builds: Multi-platform binaries (Linux, macOS, Windows)
- Publishes: GitHub Releases + Docker images (ghcr.io)
- Homebrew: Updates tap formula automatically

**Release Steps**:
```bash
# 1. Update version in relevant files
# 2. Create and push tag
git tag v1.2.3
git push origin v1.2.3

# 3. GitHub Actions automatically:
#    - Builds binaries for all platforms
#    - Creates GitHub Release with binaries
#    - Builds and pushes Docker images
#    - Updates Homebrew tap
```

## Git Workflow

**Branch naming**:
- `feature/*` - New features
- `fix/*` - Bug fixes
- `chore/*` - Maintenance tasks
- `docs/*` - Documentation updates

**Commit messages** (Conventional Commits):
- `feat:` - New feature
- `fix:` - Bug fix
- `chore:` - Maintenance (dependencies, build, etc.)
- `docs:` - Documentation
- `refactor:` - Code refactoring
- `test:` - Test updates
- `ci:` - CI/CD changes

**Protected branches**:
- `main` - Requires PR, passing checks, review

## CI/CD Workflows

### 1. Linter Workflow (`.github/workflows/linter.yml`)
- **Trigger**: Pull requests, push to main
- **Steps**: Checkout, setup Go, run `task lint` (golangci-lint)
- **Purpose**: Enforce code quality standards

### 2. Coverage Workflow (`.github/workflows/coverage-badge.yml`)
- **Trigger**: Push to main
- **Steps**: Run tests, generate coverage report, update coverage badge
- **Purpose**: Maintain test coverage visibility

### 3. Snapshot Workflow (`.github/workflows/snapshot.yml`)
- **Trigger**: Push to main (non-tags)
- **Steps**: GoReleaser snapshot build (both binaries)
- **Purpose**: Validate release process

### 4. Release Workflow (`.github/workflows/release.yml`)
- **Trigger**: Push tag `v*`
- **Steps**: GoReleaser build + publish (binaries, Docker, Homebrew)
- **Purpose**: Automated releases

## Development Tips

**Run specific tests**:
```bash
go test -v -run TestRestoreLocal ./tests/integration/
```

**Test with race detection**:
```bash
go test -race ./pkg/gitlab/
```

**Update mocks after interface changes**:
```bash
go generate ./...
```

**Build specific binary**:
```bash
go build -o gitlab-backup ./cmd/gitlab-backup/
go build -o gitlab-restore ./cmd/gitlab-restore/
```

**Run godoc server**:
```bash
task doc
# Open http://localhost:6060/pkg/
```

**Test Docker build**:
```bash
task image
```

## Debugging

**Enable verbose logging** (future enhancement):
```bash
export LOG_LEVEL=debug
./gitlab-backup ...
```

**Inspect archive contents**:
```bash
tar -tzf backup-archive.tar.gz
```

**Test S3 locally** with MinIO:
```bash
# See tests/integration/ for testcontainers examples
```

## Pre-commit Checklist

Before committing:
- [ ] Run `task lint` - No linter errors
- [ ] Run `task test` - All tests pass
- [ ] Update tests for new code
- [ ] Update CLAUDE.md if architecture changes
- [ ] Check `git status` - No unintended files
- [ ] Conventional commit message format
