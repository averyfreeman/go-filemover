# Objective
Refine the `go-filemover` repository by upgrading the documentation, enhancing the build tooling, and establishing a robust testing framework.

# Context & Guardrails
- **Environment:** This is a Go application built on Linux, specifically meant to run inside WSL2 while interacting with a Windows host.
- **Documentation Standard:** Utilize native Go 1.19+ Markdown syntax for all code docstrings. Do not use or implement third-party documentation parsers like `gdmd` or `goasciidoc`.

# Tasks

## 1. Expand and Refine `README.md`
- **Installation & Setup:** Elaborate on the steps outlined in `scripts/setup.sh`. Explain the dependency fetching, build process, and configuration setup in a user-friendly manner. 
- **Script Correction:** Please correct the invalid build command currently in `setup.sh` (change `go -o go-filemover.go` to the correct `go build` syntax referencing the `cmd/` directory).
- **Configuration Guide:** Detail the usage of `example_config.toml`. Explicitly explain the multi-glob support (comma-separated values) and define the `wd` (watched directory), `td` (target directory), and `fg` (file glob) mappings.
- **Dependencies:** Identify and document all 3rd-party libraries used in the codebase (e.g., `fsnotify`, `go-toml/v2`, `pflag`, `lipgloss`) and briefly explain their specific role in this utility.

## 2. Code Documentation Updates
- Audit the Go source files and ensure all functions, structs, and packages are fully documented using Go 1.19+ standard markdown. Generate a markdown-based docs export if a standard tool (`godoc` or `gomarkdown`) is cleanly applicable.

## 3. `Makefile` Enhancements
- Expand the current `Makefile` to include standard Go conventions.
- Add targets for `clean`, `install`, `lint` (using `golangci-lint` if possible), and `fmt`.

## 4. Unit Testing Implementation
- Write unit tests for the core logical components, specifically targeting:
  - TOML configuration parsing.
  - Multi-glob matching logic.
  - The `generateUniquePath` file collision logic.
- Implement mocked tests for filesystem operations (`os.Rename` and `io.Copy`) where feasible to ensure tests can run without requiring a live WSL/Windows boundary.

## 5. Recurring Test Protocol (CI/CD)
- Create a GitHub Actions workflow file (`.github/workflows/ci.yml`).
- Configure the workflow to trigger `make test` (and `make lint` if implemented) automatically on every `push` and `pull_request` to the main branch to ensure ongoing stability.

