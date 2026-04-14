// Package main provides a cross-platform file watching and moving utility.
package main

import (
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/pflag"
)

type TaskConfig struct {
	Wd string `toml:"wd"`
	Td string `toml:"td"`
	Fg string `toml:"fg"`
}

type Config map[string]TaskConfig

// Global application state
var (
	currentLogLevel int
	pidFilePath     string
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Fatal: Cannot determine user home directory: %v", err)
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	pidFilePath = filepath.Join(runtimeDir, "go-filemover", "go-filemover.pid")
	defaultConfigPath := filepath.Join(homeDir, ".config", "go-filemover", "config.toml")
	
	// 1. Define CLI Flags using pflag
	help := pflag.BoolP("help", "h", false, "Show this menu.")
	exitFlag := pflag.BoolP("exit", "e", false, "Gracefully shutdown background instances.")
	killFlag := pflag.BoolP("kill", "k", false, "Hard stop background instances.")
	configFlag := pflag.StringP("config", "c", defaultConfigPath, "Start using alternative configuration file.")
	debugFlag := pflag.BoolP("debug", "d", false, "Start in foreground with loglevel 7.")
	logLevelFlag := pflag.IntP("loglevel", "l", 4, "Specify loglevel Levels: 1 through 7 (7 being most verbose).")
	verboseFlag := pflag.CountP("verbose", "v", "Increase verbosity (e.g. -vv equals loglevel 6).")
	
	wdFlag := pflag.String("wd", "", "Watched Directory (Requires -td and -fg)")
	tdFlag := pflag.String("td", "", "Target Directory (Requires -wd and -fg)")
	fgFlag := pflag.String("fg", "", "File Glob (Requires -wd and -td)")

	pflag.Usage = printBeautifulHelp
	pflag.Parse()

	if *help {
		pflag.Usage()
		os.Exit(0)
	}

	// 2. Handle Process Control Flags
	if *exitFlag || *killFlag {
		manageBackgroundProcess(*exitFlag, *killFlag)
		os.Exit(0)
	}

	// 3. Resolve Log Levels
	currentLogLevel = *logLevelFlag
	if *verboseFlag > 0 {
		currentLogLevel += *verboseFlag
	}
	if *debugFlag {
		currentLogLevel = 7
	}
	if currentLogLevel > 7 {
		currentLogLevel = 7
	}

	sysLog(5, "Starting go-filemover with log level %d", currentLogLevel)

	// 4. Write PID File for future background management
	if !*debugFlag {
		writePIDFile()
		defer os.Remove(pidFilePath)
	} else {
		sysLog(7, "Debug mode active: Bypassing PID file creation.")
	}

	// 5. Configuration Resolution (CLI Overrides TOML)
	var cfg Config
	if *wdFlag != "" && *tdFlag != "" && *fgFlag != "" {
		sysLog(6, "Bypassing config file. Using inline CLI arguments.")
		cfg = Config{
			"0": {Wd: *wdFlag, Td: *tdFlag, Fg: *fgFlag},
		}
	} else {
		configData, err := os.ReadFile(*configFlag)
		if err != nil {
			log.Fatalf("Fatal: Could not read config file at %s: %v", *configFlag, err)
		}
		if err := toml.Unmarshal(configData, &cfg); err != nil {
			log.Fatalf("Fatal: Failed to parse TOML configuration: %v", err)
		}
	}

	// 6. Setup Graceful Shutdown Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		sysLog(4, "Interrupt signal received. Initiating graceful shutdown...")
		notifyWindows("go-filemover Exiting", "Service stopped safely.")
		cancel()
	}()

	var wg sync.WaitGroup

	for id, task := range cfg {
		wg.Add(1)
		
		task.Wd = os.ExpandEnv(task.Wd)
		task.Td = os.ExpandEnv(task.Td)

		// Create target directory if it does not exist
		if err := os.MkdirAll(task.Td, 0755); err != nil {
			sysLog(1, "[Task %s] Failed to create target directory %s: %v", id, task.Td, err)
			continue
		}

		initMsg := fmt.Sprintf("Watching: %s\nTarget: %s\nPattern: %s", task.Wd, task.Td, task.Fg)
		notifyWindows(fmt.Sprintf("go-filemover Ready [Task %s]", id), initMsg)

		go func(taskID string, t TaskConfig) {
			defer wg.Done()
			watchDirectory(ctx, taskID, t)
		}(id, task)
	}

	<-ctx.Done()
	wg.Wait()
	os.Exit(0)
}

// sysLog provides basic level-based logging.
// 1 = Fatal/Errors, 4 = Info, 7 = Trace/Debug
func sysLog(level int, format string, v ...interface{}) {
	if currentLogLevel >= level {
		log.Printf(format, v...)
	}
}

// printBeautifulHelp uses Lipgloss to render an aesthetically pleasing CLI menu.
func printBeautifulHelp() {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#5EEAD4")).MarginBottom(1)
	flagStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FDE047"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB"))
	noteStyle := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#9CA3AF"))

	fmt.Println(titleStyle.Render("\ngo-filemover: WSL2 Native File Watcher"))
	fmt.Println(descStyle.Render("Usage: go-filemover [OPTIONS]\n"))

	flags := []struct {
		flag string
		desc string
	}{
		{"-h, --help", "Show this menu."},
		{"-e, --exit", "Gracefully shutdown background instances."},
		{"-k, --kill", "Hard stop background instances."},
		{"-c, --config", "Start using alternative configuration file."},
		{"-d, --debug", "Start in foreground with loglevel 7."},
		{"-l, --loglevel", "Specify loglevel Levels: 1 through 7 (7 being most verbose)."},
		{"-v, --verbose", "Increase verbosity. Stackable (e.g. -vv equals loglevel 6)."},
		{"-wd, -td, -fg", "Inline configs. Must be invoked simultaneously."},
	}

	for _, f := range flags {
		fmt.Printf("  %-35s %s\n", flagStyle.Render(f.flag), descStyle.Render(f.desc))
	}
	
	fmt.Println(noteStyle.Render("\n  Note: -wd, -td, and -fg bypass the configuration file (useful for debugging)."))
	fmt.Println()
}

// manageBackgroundProcess reads the PID file and sends system signals to control background instances.
func manageBackgroundProcess(graceful bool, hardKill bool) {
	data, err := os.ReadFile(pidFilePath)
	if err != nil {
		fmt.Println("No background instance found or missing PID file.")
		return
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		fmt.Println("Invalid PID file format.")
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("Failed to find process.")
		return
	}

	if hardKill {
		fmt.Printf("Sending SIGKILL to process %d...\n", pid)
		process.Signal(syscall.SIGKILL)
	} else if graceful {
		fmt.Printf("Sending SIGTERM to process %d...\n", pid)
		process.Signal(syscall.SIGTERM)
	}
}

func writePIDFile() {
	pid := os.Getpid()
	dir := filepath.Dir(pidFilePath)
	os.MkdirAll(dir, 0755)
	os.WriteFile(pidFilePath, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func watchDirectory(ctx context.Context, taskID string, task TaskConfig) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		sysLog(1, "Error creating watcher for Task %s: %v", taskID, err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(task.Wd); err != nil {
		sysLog(1, "Error watching directory %s: %v", task.Wd, err)
		return
	}

	sysLog(4, "[Task %s] Actively watching %s", taskID, task.Wd)

	processExistingFiles(task)

	for {
		select {
		case <-ctx.Done():
			return 
		case event, ok := <-watcher.Events:
			if !ok { return }

			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				handleFileEvent(task, event.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok { return }
			sysLog(2, "[Task %s] Watcher error: %v", taskID, err)
		}
	}
}

func handleFileEvent(task TaskConfig, filePath string) {
	fileName := filepath.Base(filePath)
	
	// MULTI-GLOB MATCHING LOGIC
	patterns := strings.Split(task.Fg, ",")
	matched := false
	for _, p := range patterns {
		p = strings.TrimSpace(p) // Strip whitespace from comma delimitation
		m, err := filepath.Match(p, fileName)
		if err != nil {
			sysLog(2, "Invalid glob pattern '%s': %v", p, err)
			continue
		}
		if m {
			matched = true
			break
		}
	}

	if !matched {
		return
	}

	sysLog(7, "Glob matched for file: %s", fileName)
	time.Sleep(1 * time.Second)

	targetPath := filepath.Join(task.Td, fileName)

	if fileExists(targetPath) {
		srcCRC, err1 := calculateCRC32(filePath)
		dstCRC, err2 := calculateCRC32(targetPath)

		if err1 == nil && err2 == nil && srcCRC == dstCRC {
			sysLog(5, "Identical file exists at target. Removing source: %s", fileName)
			notifyWindows("Duplicate Ignored", fmt.Sprintf("Cleaned up duplicate: %s", fileName))
			os.Remove(filePath)
			return
		}
		
		targetPath = generateUniquePath(targetPath)
		sysLog(5, "Filename collision detected. Auto-renaming to: %s", filepath.Base(targetPath))
	}

	notifyWindows("File Detected", fmt.Sprintf("Attempting move: %s", filepath.Base(targetPath)))
	
	if err := moveFile(filePath, targetPath); err != nil {
		sysLog(1, "Failed to move %s: %v", fileName, err)
		notifyWindows("Move Failed", fmt.Sprintf("Failed to move %s: %v", fileName, err))
	} else {
		sysLog(4, "Successfully moved %s", filepath.Base(targetPath))
		notifyWindows("Move Successful", fmt.Sprintf("Moved to %s", task.Td))
	}
}

func processExistingFiles(task TaskConfig) {
	patterns := strings.Split(task.Fg, ",")
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		pattern := filepath.Join(task.Wd, p)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			sysLog(2, "Error globbing existing files: %v", err)
			continue
		}

		for _, match := range matches {
			handleFileEvent(task, match)
		}
	}
}

func calculateCRC32(filePath string) (uint32, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	hasher := crc32.NewIEEE()
	if _, err := io.Copy(hasher, f); err != nil {
		return 0, err
	}

	return hasher.Sum32(), nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func generateUniquePath(originalPath string) string {
	dir := filepath.Dir(originalPath)
	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(filepath.Base(originalPath), ext)

	counter := 1
	newPath := originalPath
	for {
		if !fileExists(newPath) {
			return newPath
		}
		newPath = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, counter, ext))
		counter++
	}
}

func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil 
	}

	if linkErr, ok := err.(*os.LinkError); ok {
		if linkErr.Err == syscall.EXDEV {
			return copyAndDelete(src, dst)
		}
	}

	return copyAndDelete(src, dst)
}

func copyAndDelete(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		srcFile.Close()
		return fmt.Errorf("failed to create destination: %w", err)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		srcFile.Close()
		dstFile.Close()
		return fmt.Errorf("failed during byte copy: %w", err)
	}

	srcFile.Close()
	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("failed to close destination file: %w", err)
	}

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("copied successfully, but failed to remove original: %w", err)
	}

	return nil
}

func getPowerShellPath() string {
	path, err := exec.LookPath("powershell.exe")
	if err == nil {
		return path
	}
	return "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"
}

func notifyWindows(title, message string) {
	safeTitle := strings.ReplaceAll(title, "'", "''")
	safeMsg := strings.ReplaceAll(message, "'", "''")

	psScript := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$balloon = New-Object System.Windows.Forms.NotifyIcon
$path = (Get-Process -id $pid).Path
$balloon.Icon = [System.Drawing.Icon]::ExtractAssociatedIcon($path)
$balloon.Visible = $True
$balloon.ShowBalloonTip(5000, '%s', '%s', 'Info')
Start-Sleep -Seconds 2
$balloon.Dispose()
`, safeTitle, safeMsg)

	psPath := getPowerShellPath()
	cmd := exec.Command(psPath, "-NoProfile", "-WindowStyle", "Hidden", "-Command", psScript)
	
	if err := cmd.Start(); err != nil {
		sysLog(1, "Notification error: failed to invoke PowerShell: %v", err)
	}
}