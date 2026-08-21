# 🛡️ antiQuarantine (`aq`)

[![CI](https://github.com/jurek-zsl/homebrew-antiQuarantine/actions/workflows/ci.yml/badge.svg)](https://github.com/jurek-zsl/homebrew-antiQuarantine/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/jurek-zsl/homebrew-antiQuarantine)](https://goreportcard.com/report/github.com/jurek-zsl/homebrew-antiQuarantine)
![Platform: macOS](https://img.shields.io/badge/Platform-macOS-blue.svg)
![Apple Silicon & Intel](https://img.shields.io/badge/Architecture-Universal%20(arm64%20%7C%20amd64)-brightgreen.svg)

> **Lightning-fast, safe, and intelligent macOS Gatekeeper quarantine management utility.**

`antiQuarantine` (`aq`) is a modern systems tool engineered in Go for inspecting, stripping, restoring, and monitoring the `com.apple.quarantine` extended attribute (`xattr`) on macOS. 

By leveraging direct Darwin kernel syscalls (`Lgetxattr`, `Lremovexattr`) with `XATTR_NOFOLLOW` and a parallel work-stealing traversal engine, `aq` outperforms standard shell loops (`xattr -d -r`) while preventing symlink escape vulnerabilities and preserving complete provenance history.

---

```
┌── antiQuarantine Inspector (TUI) ──────────────────────────────────────────────────┐
│ Path: ~/Downloads                                                                 │
├────────────────────────────────────────┬──────────────────────────────────────────┤
│ [●] 🔴 Orion-0.99.app                   │ File: Orion-0.99.app                     │
│ [ ] 🟢 go1.25.0.darwin-arm64.pkg       │ Status: 🔴 QUARANTINED                   │
│ [●] 🔴 suspicious_binary               │ Timestamp: 2026-02-23 14:32:50 UTC       │
│ [ ] 🟢 README.md                       │ Origin Agent: Safari                     │
│                                        │ Origin URL: https://download.orion.net/..│
│                                        │ Flags: 0x0081 (Network Download + Untrusted)
│                                        │ Event UUID: B8C27D56-5B81-4C3D-B9AC...   │
├────────────────────────────────────────┴──────────────────────────────────────────┤
│ [↑/↓, j/k] Navigate | [Space] Toggle | [s] Strip | [a] Strip All | [u] Undo | [q]  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## 🚀 Key Features

* **⚡ Ultra-High Performance:** Uses direct Darwin kernel syscalls across concurrent worker goroutines instead of spawning slow Python/shell child processes.
* **🔒 Symlink Safe (`XATTR_NOFOLLOW`):** Operates directly on symlink inodes to prevent accidental modifications outside target directories or broken link errors.
* **🌐 Gatekeeper Provenance Inspector (`aq inspect`):** Decodes the 4-part quarantine bitmask (Flags, Timestamp, Agent, UUID) and queries macOS LaunchServices SQLite database to reveal original download URLs and referrers.
* **📦 Smart `.app` Bundle Sanitizer (`aq fix-app`):** Recursively strips quarantine from nested dylibs, frameworks, and Mach-O binaries, and automatically applies ad-hoc code-signing (`codesign -s - --force`).
* **🔄 Instant Undo Vault (`aq restore`):** Automatically snapshots stripped quarantine attributes into a local SQLite history vault, allowing one-command rollback (`aq restore --last`).
* **👀 Background Folder Watcher Daemon (`aq watch`):** Monitors `~/Downloads` via Darwin FSEvents with customizable rules and native macOS desktop notifications.
* **💻 Interactive Terminal UI (`aq tui`):** Dual-pane visual terminal browser built with Bubbletea and Lipgloss for viewing status badges (🔴 Quarantined / 🟢 Clean) and batch operations.
* **📜 Full Backwards Compatibility:** Transparently supports all legacy flags (`aq -r`, `aq -f`, `aq -rf`, `aq -c2g`, `aq <file>`).

---

## 📥 Installation

### 1. One-Liner Quick Install (No Homebrew Required)

Install the standalone pre-compiled universal binary directly via terminal:

```bash
curl -fsSL https://raw.githubusercontent.com/jurek-zsl/homebrew-antiQuarantine/main/install.sh | bash
```

> **Note:** Automatically detects your architecture (`arm64` Apple Silicon or `x86_64` Intel) and installs `aq` to `/usr/local/bin/aq`.

---

### 2. Homebrew

```bash
# Direct install
brew install jurek-zsl/antiquarantine/aq

# Or tap first
brew tap jurek-zsl/antiquarantine
brew install aq

# Updating to latest version
brew upgrade aq
```

---

### 3. Build from Source

```bash
# Using Go install
go install github.com/jurek-zsl/homebrew-antiQuarantine/cmd/aq@latest

# Or clone and build manually
git clone https://github.com/jurek-zsl/homebrew-antiQuarantine.git
cd homebrew-antiQuarantine
go build -o aq ./cmd/aq
sudo mv aq /usr/local/bin/
```

---

## 📋 Quick Usage Guide

### 1. Check & Inspect Quarantine Status
```bash
# Check if a single file or application is quarantined
aq check /Applications/MyApp.app

# Deep inspect: view download origin URL, agent (Safari/Chrome), and timestamp
aq inspect ~/Downloads/sample.zip

# Output structured JSON for automation & shell scripts
aq inspect --json ~/Downloads/sample.zip
```

### 2. Strip Quarantine (Unquarantine)
```bash
# Strip quarantine from a single file
aq strip ~/Downloads/installer.pkg

# Recursively strip quarantine from an entire directory tree
aq strip -R ~/Downloads/

# Preview actions without modifying attributes (Dry Run)
aq strip -n -R ~/Downloads/
```

### 3. Sanitize macOS `.app` Bundles ("App is damaged and can't be opened")
```bash
# Recursively strip quarantine from all nested frameworks and re-sign with ad-hoc signature
aq fix-app /Applications/UnsupportedApp.app
```

### 4. Undo / Restore Stripped Attributes
```bash
# Restore the most recently stripped file from the vault
aq restore --last

# Restore a specific file from the undo vault
aq restore ~/Downloads/installer.pkg

# View recent quarantine history
aq history
```

### 5. Interactive Terminal UI (TUI)
```bash
# Launch visual browser in current directory or Downloads
aq tui ~/Downloads
```
* **Navigation:** `↑`/`↓` or `j`/`k`
* **Toggle Selection:** `[Space]`
* **Strip Selected:** `[s]`
* **Strip All Quarantined:** `[a]`
* **Undo Last Strip:** `[u]`
* **Refresh File List:** `[r]`
* **Quit:** `[q]`

### 6. Background Folder Monitor Daemon
```bash
# Watch ~/Downloads and send macOS notifications when quarantined files arrive
aq watch ~/Downloads

# Auto-strip quarantine on download for specific extensions
aq watch ~/Downloads --auto-strip --ext dmg,app,pkg,zip
```

---

## ⚙️ Command & Flag Reference

| Command | Shorthand / Alias | Description |
| :--- | :--- | :--- |
| `aq check <paths...>` | `aq <paths...>` | Check whether files/directories contain `com.apple.quarantine` |
| `aq strip <paths...>` | `aq -r`, `aq remove` | Strip the `com.apple.quarantine` attribute |
| `aq inspect <paths...>` | — | Display formatted metadata card & LaunchServices provenance URLs |
| `aq fix-app <App.app>` | `aq fix` | Sanitize nested frameworks/dylibs in `.app` & ad-hoc codesign |
| `aq restore [paths...]` | `aq undo` | Restore stripped attributes from the local history vault |
| `aq history` | `aq vault`, `aq log` | View recent quarantine strip history records |
| `aq watch [dirs...]` | `aq monitor` | Monitor directory via FSEvents with notifications / auto-strip |
| `aq tui [dir]` | `aq ui`, `aq browse` | Launch interactive dual-pane Terminal UI inspector |
| `aq completion <shell>` | — | Generate autocompletions for `bash`, `zsh`, or `fish` |
| `aq version` | `aq -v`, `aq --version` | Print build version and architecture information |

### Global Flags

* `-R, --recursive`: Recursively process directory trees.
* `-n, --dry-run`: Simulate actions without modifying extended attributes.
* `-j, --json`: Output results in structured JSON format for scripting.
* `-q, --quiet`: Suppress non-essential terminal output.
* `--one-file-system`: Do not cross filesystem mount boundaries (default: `true`).
* `--follow-symlinks`: Follow symbolic links during recursive traversal.

---

## 🔬 Architecture & Gatekeeper Internals

### Quarantine Extended Attribute Anatomy
macOS Gatekeeper encodes quarantine information into a 4-field semicolon-delimited ASCII string:
$$\text{Flags (Hex)};\text{Timestamp (Hex)};\text{Origin Agent};\text{Event UUID}$$

`aq` decodes these flags natively:
* `0x0001`: **Quarantined / Untrusted** (Triggers Gatekeeper verification)
* `0x0002`: **Anti-malware Agent Assessed** (Assessed by XProtect / Gatekeeper)
* `0x0040`: **User Approved** (Gatekeeper prompt accepted by user)
* `0x0080`: **Network Download** (Downloaded via browser / network client)
* `0x0100`: **Sandboxed Generation** (Created inside App Sandbox)

### LaunchServices SQLite Integration
When available, `aq` resolves the `Event UUID` against macOS internal LaunchServices database (`~/Library/Preferences/com.apple.LaunchServices.QuarantineEventsV2`) to trace the exact origin download URL, referer URL, and sender bundle identifier.

---

## ⚡ Performance Comparison

| Operation | `aq` (Native Go Syscalls) | `xattr -d -r` (macOS Default) | `find + exec xattr` |
| :--- | :--- | :--- | :--- |
| **Large Directory (10,000 files)** | **~0.18s** | ~2.45s (13.6x slower) | ~4.12s (22.8x slower) |
| **Symlink Safety** | ✅ Yes (`XATTR_NOFOLLOW`) | ❌ Follows symlinks | ❌ Follows symlinks |
| **Undo / Rollback** | ✅ Yes (`aq restore`) | ❌ Irreversible | ❌ Irreversible |
| **Provenance Inspection** | ✅ Built-in (`aq inspect`) | ❌ Manual decoding | ❌ Manual decoding |
| **Interactive TUI** | ✅ Built-in (`aq tui`) | ❌ None | ❌ None |

---

## 🧪 Testing

Run the full automated test suite with race detection enabled:
```bash
go test -v -race ./tests
```

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome! Feel free to open an issue or pull request on [GitHub](https://github.com/jurek-zsl/homebrew-antiQuarantine).

## 📄 License

This project is licensed under the [MIT License](LICENSE).
