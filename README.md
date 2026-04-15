# go-filemover

A WSL2 native file watcher and mover. This utility monitors specific directories in your WSL2 environment and automatically moves files matching defined glob patterns to a target directory (typically on the Windows host).

## Features

- **WSL2 Native:** Optimized for Linux-to-Windows filesystem interaction.
- **Multi-tasking:** Watch multiple directories with independent configurations.
- **Multi-glob Support:** Define comma-separated glob patterns for each task.
- **Collision Resolution:** Automatically renames files if the target already exists (e.g., `file (1).txt`).
- **Duplicate Detection:** Calculates CRC32 checksums to avoid moving identical files that already exist at the destination.
- **Windows Notifications:** Triggers native Windows toast notifications via PowerShell interop.

## Installation & Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/jules/go-filemover.git
   cd go-filemover
   ```

2. **Run the setup script:**
   The setup script (`scripts/setup.sh`) initializes the Go module, fetches dependencies, builds the binary, and creates a default configuration.
   ```bash
   ./scripts/setup.sh
   ```

3. **Manual Build:**
   Alternatively, you can use the `Makefile`:
   ```bash
   make build
   ```
   The binary will be located in `bin/go-filemover`.

## Configuration Guide

The application uses TOML for configuration. By default, it looks for a file at `~/.config/go-filemover/config.toml`.

### Example `config.toml`

```toml
[downloads]
wd = "/home/user/Downloads"
td = "/mnt/c/Users/User/Downloads/WSL_Sorted"
fg = "*.pdf, *.docx, *.zip"

[screenshots]
wd = "/home/user/Pictures"
td = "/mnt/c/Users/User/Pictures/WSL_Screenshots"
fg = "Screenshot_*.png"
```

### Fields

- **`wd` (Watched Directory):** The directory to monitor for new or modified files. Supports environment variables (e.g., `$HOME`).
- **`td` (Target Directory):** The destination where matched files will be moved.
- **`fg` (File Glob):** A comma-separated list of glob patterns. Files matching *any* of these patterns will be moved.

## Usage

```bash
./bin/go-filemover [OPTIONS]
```

### Options

- `-h, --help`: Show help menu.
- `-e, --exit`: Gracefully shut down background instances.
- `-k, --kill`: Hard stop background instances.
- `-c, --config`: Use an alternative configuration file.
- `-d, --debug`: Start in foreground with verbose logging (loglevel 7).
- `-l, --loglevel`: Specify log level (1-7).
- `-v, --verbose`: Increase verbosity (stackable, e.g., `-vv`).
- `-wd, -td, -fg`: Inline configuration overrides (must be used together).

## Dependencies

- **[fsnotify](https://github.com/fsnotify/fsnotify):** Provides cross-platform filesystem notifications.
- **[go-toml/v2](https://github.com/pelletier/go-toml):** High-performance TOML parser.
- **[pflag](https://github.com/spf13/pflag):** POSIX-compliant flags for Go.
- **[lipgloss](https://github.com/charmbracelet/lipgloss):** Style definitions for nice terminal output.

## Development

- **Build:** `make build`
- **Test:** `make test`
- **Format:** `make fmt`
- **Lint:** `make lint`
