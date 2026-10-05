# Changelog

All notable changes to DevGit will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-10-05

### Added
- **Release & Distribution (Phase 12)**:
  - Version command (`devgit version`, `devgit version --short`) with compile-time linker flag injection (`-ldflags`).
  - Shell completion scripts for `bash`, `zsh`, `fish`, and `powershell` via `devgit completion <shell>`.
  - GoReleaser configuration (`.goreleaser.yaml`) supporting cross-platform release archives (`.tar.gz` and `.zip`).
  - Cross-platform binary targets: macOS (`darwin/amd64`, `darwin/arm64`), Linux (`linux/amd64`, `linux/arm64`), Windows (`windows/amd64`, `windows/arm64`).
  - GitHub Actions CI workflow (`.github/workflows/ci.yml`) for multi-OS testing (Ubuntu, macOS, Windows).
  - GitHub Actions release workflow (`.github/workflows/release.yml`) automated upon `v*` tag creation.
  - Release check verification script (`scripts/release-check.sh`).
  - Comprehensive documentation for installation (`go install`, GitHub Releases, package managers, macOS Gatekeeper), architecture, and verification.
- **SQLite Application Metadata (Phase 10)**:
  - Local metadata store powered by pure-Go SQLite (`modernc.org/sqlite`) without CGO.
  - Persistent command execution history with duration and status tracking (`devgit history`, `devgit history clear`).
  - Database status inspection and integrity check (`devgit db status`).
  - User and TUI preference persistence (`devgit db pref`).
  - Safe database failure mode: DevGit continues functioning even if SQLite is disabled or unavailable.
- **Hierarchical Configuration (Phase 9)**:
  - Layered configuration resolution: CLI flags > environment variables > local config > global config > defaults.
  - Config management commands: `devgit config list`, `devgit config get`, `devgit config set`, `devgit config path`.
- **Bubble Tea Terminal UI (Phase 8)**:
  - Interactive terminal dashboard built with Bubble Tea and Lip Gloss.
  - Keyboard-driven views for status, branches, commits, diffs, and stashes.
- **Stash Management (Phase 7)**:
  - Safe stash workflows: `devgit stash`, `devgit stash list`, `devgit stash show`, `devgit stash apply`, `devgit stash pop`, `devgit stash drop`.
- **Branch Management (Phase 6)**:
  - Safe branch commands: `devgit branch`, `devgit branch list`, `devgit branch create`, `devgit branch switch`, `devgit branch delete`.
  - Safeguards against deleting current or protected branches, and warnings on uncommitted changes.
- **Push & Pull (Phase 5)**:
  - Safe upstream synchronization (`devgit push`, `devgit pull`) with upstream branch tracking and confirmation checks.
- **Commit Workflow (Phase 4)**:
  - Interactive and flags-based commit workflows (`devgit commit`).
- **Diff & Log (Phase 3)**:
  - Unified and styled diff view (`devgit diff`) and structured commit log viewer (`devgit log`).
- **Git Foundation (Phase 2)**:
  - Safe Git client abstraction using `exec.Command` argument arrays without shell interpolation or `sh -c`.
- **Project Foundation (Phase 1)**:
  - Core CLI scaffolding using Cobra, status command (`devgit status`), and repository detection.

### Note on Phase 11
- Phase 11 (AI Assistant) was intentionally skipped. DevGit remains a completely local-first, zero-telemetry Git assistant with no external AI or cloud API dependencies.
