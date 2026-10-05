# DevGit 🚀

> A safer, developer-friendly Git assistant with a powerful CLI and interactive terminal UI.

DevGit is designed not to replace Git, but to provide a safer, easier, and more productive interface on top of standard Git workflows. It pairs an ergonomic command-line interface with an interactive keyboard-driven terminal dashboard, local command history, and conservative safeguards against accidental data loss.

---

## Features

* **Git Status**: Clear, actionable overview of staged, unstaged, and untracked changes.
* **Commit Workflow**: Interactive and guided commit workflows with selective staging guards.
* **Diff Viewer**: Unified and styled diff views for working tree, index, and commit comparisons.
* **Commit Log**: Formatted, searchable commit history with rich metadata.
* **Safe Push**: Remote synchronization with ahead/behind tracking, upstream branch discovery, and protected branch safeguards.
* **Safe Pull**: Fast-forward only updates with dirty working-tree protection and rebase/merge conflict detection.
* **Branch Management**: Intuitive branch listing, creation, safe switching, guarded deletion, and renaming.
* **Stash Management**: Non-destructive stash stack management with apply, pop, and guarded drop commands.
* **Interactive TUI**: Polished, keyboard-driven dashboard powered by Bubble Tea and Lip Gloss.
* **Layered Configuration**: Flexible YAML configuration with CLI flags, environment variables, local repository overrides, and global user preferences.
* **SQLite Metadata**: Embedded, zero-configuration local database for command history and persistent UI preferences.
* **Command History**: Execution auditing with runtime metrics and success/failure tracking.
* **Cross-Platform Support**: Reproducible, pure-Go cross-platform binaries for macOS (Intel & Apple Silicon), Linux (AMD64 & ARM64), and Windows (AMD64 & ARM64).

---

## Architecture

```text
                DevGit CLI
                    │
          ┌─────────┴─────────┐
          │                   │
        Cobra             Bubble Tea
          │                   │
          └─────────┬─────────┘
                    │
               Application
                    │
          ┌─────────┴─────────┐
          │                   │
     internal/git      internal/database
          │                   │
          ↓                   ↓
       Git CLI              SQLite
```

### Architectural Guarantees

* **Git is the Source of Truth**: All Git state (commits, tree objects, branches, tags, remotes, refs, and working trees) is managed exclusively by the official Git CLI.
* **SQLite is Metadata-Only**: The embedded SQLite database (`internal/database`) stores DevGit's own operational metadata only (command execution duration, status history, and UI preferences). It **never** stores Git objects, diffs, file contents, credentials, or sensitive repository data.
* **Non-Blocking SQLite**: If the SQLite database is unavailable, disabled, or locked, DevGit continues functioning normally without interrupting any Git operations.
* **Safe Execution**: All external commands are executed using structured argument arrays (`exec.Command`) without shell interpretation (`sh -c` or `bash -c`), preventing command injection.

---

## Installation

### Option A — Prebuilt Binaries (GitHub Releases)

Download precompiled binaries directly from [GitHub Releases](https://github.com/animesh67samanta/devgit/releases/latest).

| Operating System | Architecture | Archive |
| :--- | :--- | :--- |
| **macOS** (Apple Silicon M1/M2/M3/M4) | `arm64` | `devgit_<version>_darwin_arm64.tar.gz` |
| **macOS** (Intel) | `amd64` | `devgit_<version>_darwin_amd64.tar.gz` |
| **Linux** (64-bit x86) | `amd64` | `devgit_<version>_linux_amd64.tar.gz` |
| **Linux** (64-bit ARM) | `arm64` | `devgit_<version>_linux_arm64.tar.gz` |
| **Windows** (64-bit x86) | `amd64` | `devgit_<version>_windows_amd64.zip` |
| **Windows** (64-bit ARM) | `arm64` | `devgit_<version>_windows_arm64.zip` |

#### Verifying Release Checksums

Every release includes a `checksums.txt` file containing SHA256 hashes of all release archives.

On macOS and Linux:
```bash
# Download archive and checksums
curl -sLO https://github.com/animesh67samanta/devgit/releases/download/v1.0.0/devgit_1.0.0_darwin_arm64.tar.gz
curl -sLO https://github.com/animesh67samanta/devgit/releases/download/v1.0.0/checksums.txt

# Verify checksum
shasum -a 256 -c checksums.txt --ignore-missing
# Or on Linux:
sha256sum -c checksums.txt --ignore-missing
```

On Windows (PowerShell):
```powershell
Get-FileHash .\devgit_1.0.0_windows_amd64.zip -Algorithm SHA256
# Compare the output hash with the entry in checksums.txt
```

---

### Option B — Go Install

If you have Go installed, install directly into your `$GOPATH/bin`:

```bash
go install github.com/animesh67samanta/devgit@latest
```

Or build from source:

```bash
git clone https://github.com/animesh67samanta/devgit.git
cd devgit
go build -o devgit .
```

---

### Option C — Package Manager (Roadmap)

Homebrew distribution is planned via a custom tap:
```bash
# Future roadmap:
brew install animesh67samanta/tap/devgit
```
*(Package manager formulas will be published alongside formal registry setup).*

---

## Operating System Setup

### macOS Installation

1. Download the archive for your architecture (`arm64` for Apple Silicon or `amd64` for Intel).
2. Extract the binary:
   ```bash
   tar -xzf devgit_1.0.0_darwin_arm64.tar.gz
   chmod +x devgit
   ```
3. Install to your PATH:
   ```bash
   # System-wide installation (requires sudo):
   sudo mv devgit /usr/local/bin/devgit

   # Or user-local installation (recommended, no sudo required):
   mkdir -p ~/.local/bin
   mv devgit ~/.local/bin/devgit
   # Ensure ~/.local/bin is in your PATH in ~/.zshrc or ~/.bash_profile:
   # export PATH="$HOME/.local/bin:$PATH"
   ```

#### macOS Gatekeeper & Quarantine Notice
Because DevGit binaries downloaded through a browser are quarantined by default, macOS may display a notice stating *"devgit cannot be opened because the developer cannot be verified"*.

To safely clear the quarantine attribute without disabling system security:
```bash
xattr -d com.apple.quarantine /path/to/devgit
```

---

### Linux Installation

1. Download the archive for your architecture (`amd64` or `arm64`).
2. Extract the binary:
   ```bash
   tar -xzf devgit_1.0.0_linux_amd64.tar.gz
   chmod +x devgit
   ```
3. Install to your PATH:
   ```bash
   # System-wide installation:
   sudo mv devgit /usr/local/bin/devgit

   # Or user-local installation:
   mkdir -p ~/.local/bin
   mv devgit ~/.local/bin/devgit
   # Ensure ~/.local/bin is in your PATH in ~/.bashrc or ~/.zshrc:
   # export PATH="$HOME/.local/bin:$PATH"
   ```

---

### Windows Installation

1. Download the `.zip` archive for Windows (`amd64` or `arm64`).
2. Extract `devgit.exe` to a permanent location, e.g., `C:\Program Files\DevGit\` or `%LOCALAPPDATA%\DevGit\`.
3. Add the directory to your user `PATH`:
   - In PowerShell:
     ```powershell
     [Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:LOCALAPPDATA\DevGit", "User")
     ```
   - Or open **System Properties** → **Environment Variables** → Select `Path` → Click **Edit** → **New** → Add the path where `devgit.exe` is located.
4. Open a new PowerShell or Command Prompt terminal and run:
   ```powershell
   devgit version
   ```

---

## Shell Completion

DevGit includes built-in autocompletion script generation for Bash, Zsh, Fish, and PowerShell.

### Bash

```bash
# Load in current session
source <(devgit completion bash)

# Load automatically for every session (Linux):
devgit completion bash | sudo tee /etc/bash_completion.d/devgit > /dev/null

# Load automatically on macOS with Homebrew bash-completion:
devgit completion bash > $(brew --prefix)/etc/bash_completion.d/devgit
```

### Zsh

```bash
# Ensure completion is enabled in ~/.zshrc:
echo "autoload -U compinit; compinit" >> ~/.zshrc

# Generate completion script into your completion path:
devgit completion zsh > "${fpath[1]}/_devgit"
```

### Fish

```bash
# Load in current session
devgit completion fish | source

# Persist across sessions:
mkdir -p ~/.config/fish/completions
devgit completion fish > ~/.config/fish/completions/devgit.fish
```

### PowerShell

```powershell
# Load in current session
devgit completion powershell | Out-String | Invoke-Expression

# Persist across sessions:
Add-Content -Path $PROFILE -Value (devgit completion powershell | Out-String)
```

---

## Usage

### Overview & Subcommands

```bash
# Application version and build information
devgit version
devgit version --short

# General help and command list
devgit --help

# Working tree status
devgit status

# Inspect changes (structured summary or raw patch)
devgit diff
devgit diff --staged
devgit diff --raw

# Commit history
devgit log
devgit log -n 10

# Guided, safe commit workflow
devgit commit
devgit commit -m "commit message" -y

# Safe remote push
devgit push
devgit push -y

# Safe fast-forward pull
devgit pull
devgit pull -y

# Branch management
devgit branch
devgit branch list
devgit branch create <name> [-s]
devgit branch switch <name> [-y]
devgit branch delete <name> [-f] [-y]
devgit branch rename [old] <new>

# Stash management
devgit stash
devgit stash list
devgit stash save [message] [-u] [-y]
devgit stash pop [index] [-y]
devgit stash apply [index] [-y]
devgit stash drop [index] [-y]

# Interactive Terminal UI
devgit tui

# Configuration management
devgit config
devgit config list [--local | --global]
devgit config get <key>
devgit config set [--local] <key> <value>
devgit config reset [--local] <key>

# Command execution history
devgit history
devgit history list [--limit N] [--success | --failed]
devgit history clear [--keep N]

# SQLite metadata database diagnostics
devgit db status
devgit db repos
devgit db prune [--keep N]
devgit db pref list
devgit db pref get <key>
devgit db pref set <key> <value>
```

---

### Terminal UI

Launch the keyboard-driven interactive terminal dashboard:

```bash
devgit tui
```

#### Keyboard Shortcuts

| Key | Action |
| :--- | :--- |
| `1` .. `6` | Direct jump to view (`1`: Dashboard, `2`: Status, `3`: Branches, `4`: Log, `5`: Diff, `6`: Stashes) |
| `Tab` / `→` / `l` | Next view tab |
| `Shift+Tab` / `←` / `h` | Previous view tab |
| `↑` / `k`, `↓` / `j` | Navigate list items |
| `Enter` | Select / inspect item details |
| `p` | Safe push (with ahead check & confirmation) |
| `P` / `u` | Safe fast-forward pull (with behind check) |
| `r` | Refresh repository state |
| `?` | Toggle help overlay |
| `Esc` | Cancel prompt / close modal |
| `q` / `Ctrl+C` | Quit DevGit TUI |

---

## Configuration

DevGit supports a clean, multi-tiered configuration system. It works out of the box with zero configuration required.

### Hierarchy & Precedence

Settings are resolved in strict order of priority (highest wins):

```text
CLI flags (--color, --no-color)
    ↓
Environment variables (DEVGIT_*)
    ↓
Repository/local configuration (.git/devgit.yaml)
    ↓
Global configuration (config.yaml)
    ↓
Built-in defaults
```

### Locations

* **macOS**: `~/Library/Application Support/devgit/config.yaml`
* **Linux**: `$XDG_CONFIG_HOME/devgit/config.yaml` (fallback: `~/.config/devgit/config.yaml`)
* **Windows**: `%AppData%\devgit\config.yaml`
* **Repository-Local**: `.git/devgit.yaml`

---

## SQLite Metadata Database

DevGit incorporates a lightweight, embedded SQLite metadata database for operational tracking.

### Locations

* **macOS**: `~/Library/Application Support/devgit/devgit.db`
* **Linux**: `$XDG_DATA_HOME/devgit/devgit.db` (fallback: `~/.local/share/devgit/devgit.db`)
* **Windows**: `%AppData%\devgit\devgit.db`

---

## Release Process

DevGit releases are automated through GitHub Actions and GoReleaser.

### 1. Pre-Flight Verification

Run the local release validation suite:
```bash
./scripts/release-check.sh
```

This verifies formatting (`gofmt`), static analysis (`go vet`), unit tests, the Go race detector, native compilation, and cross-platform compilation across all 6 target architectures.

### 2. Creating a Release Tag

```bash
# Ensure working branch is clean and up to date
git checkout main
git pull

# Create annotated semantic version tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push tag to trigger release workflow
git push origin v1.0.0
```

### 3. Automated Release Pipeline

Pushing a `v*` tag triggers `.github/workflows/release.yml`, which:
1. Runs `go vet`, unit tests, and race detection.
2. Invokes GoReleaser to cross-compile binaries with embedded version metadata.
3. Packages `.tar.gz` and `.zip` archives with `README.md`, `LICENSE`, and `CHANGELOG.md`.
4. Computes SHA256 hashes into `checksums.txt`.
5. Publishes the release and uploads artifacts to GitHub Releases.

---

## Project Roadmap

```text
Phase 1  — Foundation              ✅
Phase 2  — Git Foundation          ✅
Phase 3  — Diff & Log              ✅
Phase 4  — Commit Workflow         ✅
Phase 5  — Push & Pull             ✅
Phase 6  — Branch Management       ✅
Phase 7  — Stash Management        ✅
Phase 8  — Bubble Tea TUI          ✅
Phase 9  — Configuration           ✅
Phase 10 — SQLite                  ✅
Phase 11 — AI Assistant            ⏭ Skipped
Phase 12 — Release & Distribution  ✅
```

---

## License

This project is licensed under the [MIT License](LICENSE).
