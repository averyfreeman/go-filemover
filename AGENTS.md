# Jules Agent Guidelines

## Architecture Context
* **Environment:** This is a Linux binary specifically designed to run inside WSL2, but it interacts with a Windows host.
* **OS Interop:** We intentionally invoke `powershell.exe` from Go to trigger Windows native toast notifications. **Do not** attempt to refactor this to use Linux-native notification libraries (like `notify-send` or `dbus`).
* **Filesystem Boundaries:** We intentionally catch `syscall.EXDEV` (Invalid cross-device link) when using `os.Rename` to handle moves between WSL's ext4 and Windows' NTFS. Do not optimize this fallback away.

## Execution & Testing
* To build: `make build`
* To test: `make test`

## Review
Commit to new branch before requesting a PR.
