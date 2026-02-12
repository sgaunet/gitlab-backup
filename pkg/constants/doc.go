// Package constants provides centralized configuration constants for the gitlab-backup project.
//
// This package consolidates all hard-coded values, timeouts, limits, and magic numbers
// from across the codebase into a single, well-documented source of truth.
//
// Organization:
//   - gitlab.go: GitLab API constants (rate limits, timeouts, endpoints)
//   - storage.go: Storage constants (buffer sizes, file permissions)
//   - validation.go: Validation constraints (AWS limits, config boundaries)
//   - output.go: CLI output formatting constants
//
// Modifying Constants:
// Most constants are based on external API limits (GitLab, AWS) or performance
// characteristics. Before modifying:
//  1. Check the documentation comment for rationale
//  2. Verify external API documentation (links provided)
//  3. Test thoroughly with the new value
//  4. Update related constants if needed
//
// See each file for detailed documentation on specific constant groups.
package constants
