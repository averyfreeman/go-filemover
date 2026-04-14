// Package main provides the entry point for the go-filemover utility.
// go-filemover is a WSL2 native file watcher that moves files to a target directory (often on the Windows host) based on glob patterns.
package main

import (
	"context"
	"fmt"
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
	"github.com/jules/go-filemover/internal/config"
	"github.com/jules/go-filemover/internal/fs"
	"github.com/jules/go-filemover/internal/mover"
	"github.com/spf13/pflag"
)

const pidFilePath = "/tmp/go-filemover.pid"

var (
	currentLogLevel = 4
	fileMover       *mover.Mover
	filesystem      fs.FileSystem
)

func init() {
	filesystem = fs.OSFileSystem{}
	fileMover = mover.NewMover(filesystem)
}

func main() {
	// 1. Define CLI Flags
	helpFlag := pflag.BoolP("help", "h", false, "Show help menu")
	exitFlag := pflag.BoolP("exit", "e", false, "Gracefully stop background instances")
	killFlag := pflag.BoolP("kill", "k", false, "Force stop background instances")
	configFlag := pflag.StringP("config", "c", "config.toml", "Path to configuration file")
	debugFlag := pflag.BoolP("debug", "d", false, "Enable debug mode (loglevel 7, foreground)")
	logLevelFlag := pflag.IntP("loglevel", "l", 4, "Set log level (1-7)")
	verboseFlag := pflag.CountP("verbose", "v", "Increase verbosity")

	// Inline task overrides
	wdFlag := pflag.String("wd", "", "Watched directory (inline override)")
	tdFlag := pflag.String("td", "", "Target directory (inline override)")
	fgFlag := pflag.String("fg", "", "File glob pattern (inline override)")

	pflag.Parse()

	// 2. Handle Help & Process Management
	if *helpFlag {
		printBeautifulHelp()
		return
	}

	if *exitFlag || *killFlag {
		manageBackgroundProcess(*exitFlag, *killFlag)
		return
	}

	// 3. Set Log Level
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
	var cfg config.Config
	if *wdFlag != "" && *tdFlag != "" && *fgFlag != "" {
		sysLog(6, "Bypassing config file. Using inline CLI arguments.")
		cfg = config.Config{
			"0": {Wd: *wdFlag, Td: *tdFlag, Fg: *fgFlag},
		}
	} else {
		var err error
		cfg, err = config.LoadConfig(*configFlag)
		if err != nil {
			log.Fatalf("Fatal: Could not load configuration: %v", err)
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

		task.ExpandPaths()

		// Create target directory if it does not exist
		if err := filesystem.MkdirAll(task.Td, 0755); err != nil {
			sysLog(1, "[Task %s] Failed to create target directory %s: %v", id, task.Td, err)
			continue
		}

		initMsg := fmt.Sprintf("Watching: %s\nTarget: %s\nPattern: %s", task.Wd, task.Td, task.Fg)
		notifyWindows(fmt.Sprintf("go-filemover Ready [Task %s]", id), initMsg)

		go func(taskID string, t config.TaskConfig) {
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
	filesystem.MkdirAll(dir, 0755)
	os.WriteFile(pidFilePath, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func watchDirectory(ctx context.Context, taskID string, task config.TaskConfig) {
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
			if !ok {
				return
			}

			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				handleFileEvent(task, event.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			sysLog(2, "[Task %s] Watcher error: %v", taskID, err)
		}
	}
}

func handleFileEvent(task config.TaskConfig, filePath string) {
	fileName := filepath.Base(filePath)

	patterns := task.GetPatterns()
	matched, err := fileMover.MatchGlob(patterns, fileName)
	if err != nil {
		sysLog(2, "Glob matching error: %v", err)
		return
	}

	if !matched {
		return
	}

	sysLog(7, "Glob matched for file: %s", fileName)
	time.Sleep(1 * time.Second)

	targetPath := filepath.Join(task.Td, fileName)

	if fileMover.FileExists(targetPath) {
		srcCRC, err1 := fileMover.CalculateCRC32(filePath)
		dstCRC, err2 := fileMover.CalculateCRC32(targetPath)

		if err1 == nil && err2 == nil && srcCRC == dstCRC {
			sysLog(5, "Identical file exists at target. Removing source: %s", fileName)
			notifyWindows("Duplicate Ignored", fmt.Sprintf("Cleaned up duplicate: %s", fileName))
			filesystem.Remove(filePath)
			return
		}

		targetPath = fileMover.GenerateUniquePath(targetPath)
		sysLog(5, "Filename collision detected. Auto-renaming to: %s", filepath.Base(targetPath))
	}

	notifyWindows("File Detected", fmt.Sprintf("Attempting move: %s", filepath.Base(targetPath)))

	if err := fileMover.MoveFile(filePath, targetPath); err != nil {
		sysLog(1, "Failed to move %s: %v", fileName, err)
		notifyWindows("Move Failed", fmt.Sprintf("Failed to move %s: %v", fileName, err))
	} else {
		sysLog(4, "Successfully moved %s", filepath.Base(targetPath))
		notifyWindows("Move Successful", fmt.Sprintf("Moved to %s", task.Td))
	}
}

func processExistingFiles(task config.TaskConfig) {
	patterns := task.GetPatterns()
	for _, p := range patterns {
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
