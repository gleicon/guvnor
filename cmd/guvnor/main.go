package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/gleicon/guvnor/internal/cert"
	"github.com/gleicon/guvnor/internal/client"
	"github.com/gleicon/guvnor/internal/config"
	"github.com/gleicon/guvnor/internal/discovery"
	"github.com/gleicon/guvnor/internal/env"
	"github.com/gleicon/guvnor/internal/logs"
	"github.com/gleicon/guvnor/internal/process"
	"github.com/gleicon/guvnor/internal/procfile"
	"github.com/gleicon/guvnor/internal/server"
	"github.com/gleicon/guvnor/pkg/logger"
)

var (
	configFile string
	log        *logrus.Logger
	version    = "dev"
	daemon     bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		if strings.Contains(err.Error(), "config") {
			fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Try: guvnor init\n")
		} else if strings.Contains(err.Error(), "permission") {
			fmt.Fprintf(os.Stderr, "Permission denied: %v\n", err)
			fmt.Fprintf(os.Stderr, "Try: sudo guvnor or check file permissions\n")
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "guvnor",
	Short: "Guv'nor - Process manager with reverse proxy and TLS",
	Long: `Guv'nor is both a CLI tool and server for managing application processes.
It provides reverse proxy with automatic TLS certificates and process lifecycle management.

Basic workflow:
  guvnor init      # Setup everything (Procfile, .env, config)
  guvnor start     # Start server and all processes
  guvnor stop      # Stop all processes
  guvnor logs      # View process logs
  guvnor shell     # Interactive management
  guvnor validate  # Check configuration`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Opinionated init command - sets up everything
var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize application (Procfile, .env, config)",
	Long: `Opinionated initialization that creates everything needed:
- Detects applications in current directory
- Creates Procfile with detected processes
- Creates .env template with sensible defaults
- Creates minimal guvnor.yaml config
- Sets up .gitignore entries`,
	Args: cobra.MaximumNArgs(1),
	Run:  runInit,
}

// Server/daemon mode
var startCmd = &cobra.Command{
	Use:   "start [app-name]",
	Short: "Start server and all apps, or specific app",
	Long: `Start apps:
- start             # Start server and all apps
- start web-app     # Start server and only 'web-app'
- start --daemon    # Run in daemon mode`,
	Args: cobra.MaximumNArgs(1),
	Run:  runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop [app-name]",
	Short: "Stop all processes or specific app gracefully",
	Long: `Stop processes:
- stop              # Stop all apps
- stop web-app      # Stop only the 'web-app' process`,
	Args: cobra.MaximumNArgs(1),
	Run:  runStop,
}

var restartCmd = &cobra.Command{
	Use:   "restart [app-name]",
	Short: "Restart all processes or specific app",
	Long: `Restart processes:
- restart           # Restart all apps
- restart api-service # Restart only the 'api-service' process`,
	Args: cobra.MaximumNArgs(1),
	Run:  runRestart,
}

var logsCmd = &cobra.Command{
	Use:   "logs [app-name]",
	Short: "Show app logs",
	Long: `Show logs from apps:
- logs               # Show all app logs (interleaved)
- logs web-app       # Show logs from 'web-app' only
- logs -f api-service # Follow logs from 'api-service'`,
	Args: cobra.MaximumNArgs(1),
	Run:  runLogs,
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Interactive process management shell",
	Long: `Interactive shell for managing processes.
Available commands: status, start, stop, restart, logs, ps, help, quit`,
	Run: runShell,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration and environment",
	Long: `Validates:
- Procfile syntax and processes
- Environment variables and .env files
- Configuration consistency
- Port conflicts and dependencies`,
	Run: runValidate,
}

var statusCmd = &cobra.Command{
	Use:   "status [app-name]",
	Short: "Show status of all apps or specific app",
	Long: `Show app status:
- status             # Show status of all apps
- status web-app     # Show detailed status of 'web-app' only`,
	Args: cobra.MaximumNArgs(1),
	Run:  runStatus,
}

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Certificate management commands",
	Long: `Manage TLS certificates for your applications:
- cert info    # Show certificate information
- cert renew   # Renew expiring certificates
- cert cleanup # Clean up expired certificates`,
}

var certInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show certificate information",
	Run:   runCertInfo,
}

var certRenewCmd = &cobra.Command{
	Use:   "renew",
	Short: "Renew expiring certificates",
	Run:   runCertRenew,
}

var certCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up expired certificates",
	Run:   runCertCleanup,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file path")
	rootCmd.PersistentFlags().Bool("debug", false, "debug logging")
	rootCmd.PersistentFlags().Bool("quiet", false, "minimal output")

	// Start command flags
	startCmd.Flags().BoolVar(&daemon, "daemon", false, "run as daemon")
	startCmd.Flags().String("domain", "", "domain for TLS certificates")
	startCmd.Flags().String("email", "", "email for Let's Encrypt")
	startCmd.Flags().Bool("dev", false, "development mode (HTTP only)")

	// Logs command flags
	logsCmd.Flags().BoolP("follow", "f", false, "follow logs")
	logsCmd.Flags().IntP("lines", "n", 100, "number of lines to show")

	// Init command flags
	initCmd.Flags().Bool("force", false, "overwrite existing files")
	initCmd.Flags().Bool("minimal", false, "create minimal configuration")

	viper.BindPFlags(rootCmd.PersistentFlags())
	viper.BindPFlags(startCmd.Flags())
	viper.BindPFlags(logsCmd.Flags())
	viper.BindPFlags(initCmd.Flags())

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(statusCmd)
	
	// Certificate management commands
	certCmd.AddCommand(certInfoCmd)
	certCmd.AddCommand(certRenewCmd)
	certCmd.AddCommand(certCleanupCmd)
	rootCmd.AddCommand(certCmd)
}

func initConfig() {
	log = logger.New(viper.GetBool("debug"))

	// Set up global log manager and hook to capture all logs
	globalLogManager := logs.GetGlobalLogManager()
	logHook := logs.NewLogManagerHook(globalLogManager)
	log.AddHook(logHook)

	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("guvnor")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("/etc/guvnor")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("GUVNOR")

	// Don't fail if no config file
	viper.ReadInConfig()
}

// Command implementations

func runInit(cmd *cobra.Command, args []string) {
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	force := viper.GetBool("force")
	minimal := viper.GetBool("minimal")

	fmt.Printf("Initializing Guv'nor in: %s\n", targetDir)

	// 1. Detect applications
	fmt.Println("Detecting applications...")
	apps, err := discovery.DiscoverApps(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to detect applications: %v\n", err)
		os.Exit(1)
	}

	if len(apps) > 0 {
		fmt.Printf("Found %d applications:\n", len(apps))
		for _, app := range apps {
			fmt.Printf("  - %s (%s)\n", app.Name, app.Type)
		}
	} else {
		fmt.Println("No applications detected, creating minimal setup")
	}

	// 2. Create Procfile
	procfilePath := targetDir + "/Procfile"
	if !fileExists(procfilePath) || force {
		if len(apps) > 0 {
			if err := procfile.CreateSmartProcfile(procfilePath, apps); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create Procfile: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Created: %s\n", procfilePath)
		} else {
			if err := procfile.CreateEmptyProcfile(procfilePath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create Procfile: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Created: %s (empty template)\n", procfilePath)
		}
	} else {
		fmt.Printf("Exists: %s\n", procfilePath)
	}

	// 3. Create .env file
	envPath := targetDir + "/.env"
	if !fileExists(envPath) || force {
		if err := env.CreateSampleEnvFile(envPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create .env: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created: %s\n", envPath)
	} else {
		fmt.Printf("Exists: %s\n", envPath)
	}

	// 4. Create guvnor.yaml config
	configPath := targetDir + "/guvnor.yaml"
	if !fileExists(configPath) || force {
		cfg := createSmartConfig(apps, minimal)
		if err := config.WriteConfig(cfg, configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created: %s\n", configPath)
	} else {
		fmt.Printf("Exists: %s\n", configPath)
	}

	// 5. Update .gitignore
	gitignorePath := targetDir + "/.gitignore"
	if err := updateGitignore(gitignorePath); err != nil {
		fmt.Printf("Warning: Could not update .gitignore: %v\n", err)
	} else {
		fmt.Printf("Updated: %s\n", gitignorePath)
	}

	fmt.Println("\nInitialization complete!")
	fmt.Println("Next steps:")
	fmt.Println("  1. Review and edit Procfile, .env, and guvnor.yaml")
	fmt.Println("  2. Run: guvnor validate")
	fmt.Println("  3. Run: guvnor start")
}

func runStart(cmd *cobra.Command, args []string) {
	fmt.Println("Starting Guv'nor server...")

	// Load configuration
	pf, err := loadProcfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load Procfile: %v\n", err)
		os.Exit(1)
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Create server
	srv := server.New(cfg, pf, log)

	// Handle daemon mode
	if daemon {
		fmt.Println("Running as daemon...")
		// Simple daemonization: detach from terminal
		if os.Getppid() != 1 {
			// Fork and exit parent
			fmt.Println("Forking to background...")
			os.Exit(0)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server started successfully")
	fmt.Printf("Processes: %d\n", len(pf.Processes))
	fmt.Println("Press Ctrl+C to stop")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	cancel()

	if err := srv.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Shutdown complete")
}

func runStop(cmd *cobra.Command, args []string) {
	var appName string
	if len(args) > 0 {
		appName = args[0]
		fmt.Printf("Stopping app: %s...\n", appName)
	} else {
		fmt.Println("Stopping all processes...")
	}

	// Try to connect to running server via API
	port, err := client.DetectServerPort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure guvnor server is running with: guvnor start\n")
		os.Exit(1)
	}

	apiClient := client.NewClient(port)
	
	if appName != "" {
		// TODO: Implement app-specific stop via API
		fmt.Printf("App-specific stop not yet implemented for %s\n", appName)
		fmt.Println("Use 'guvnor stop' to stop all apps for now")
		return
	}
	
	results, err := apiClient.StopProcesses()
	
	if len(results) == 0 {
		fmt.Println("No running processes found")
		return
	}

	// Display detailed stop results
	fmt.Printf("\n%-15s %-8s %-10s %-8s %s\n", "PROCESS", "PID", "STATUS", "TIME", "DETAILS")
	fmt.Printf("%-15s %-8s %-10s %-8s %s\n", "-------", "---", "------", "----", "-------")
	
	for _, result := range results {
		pidStr := "-"
		if result.PID > 0 {
			pidStr = fmt.Sprintf("%d", result.PID)
		}
		
		durationStr := "-"
		if result.Duration > 0 {
			durationStr = fmt.Sprintf("%.1fs", result.Duration.Seconds())
		}
		
		details := ""
		if result.Error != nil {
			details = result.Error.Error()
			if len(details) > 40 {
				details = details[:37] + "..."
			}
		}
		
		// Color code status
		var statusDisplay string
		switch result.Status {
		case "stopped":
			statusDisplay = "\033[32mstopped\033[0m"   // Green
		case "killed":
			statusDisplay = "\033[33mkilled\033[0m"    // Yellow
		case "error":
			statusDisplay = "\033[31merror\033[0m"     // Red
		case "not_running":
			statusDisplay = "\033[90mnot_run\033[0m"   // Gray
		default:
			statusDisplay = result.Status
		}
		
		fmt.Printf("%-15s %-8s %-18s %-8s %s\n", 
			result.Name, pidStr, statusDisplay, durationStr, details)
	}
	
	if err != nil {
		fmt.Printf("\nWarning: Some processes could not be stopped: %v\n", err)
	} else {
		fmt.Println("\nAll processes stopped successfully")
	}
}

func runRestart(cmd *cobra.Command, args []string) {
	pm := process.NewManager(log)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if len(args) > 0 {
		fmt.Printf("Restarting process: %s\n", args[0])
		if err := pm.Restart(ctx, args[0]); err != nil {
			fmt.Printf("Error restarting %s: %v\n", args[0], err)
		}
	} else {
		fmt.Println("Restarting all processes...")
		// Stop all then start all
		runStop(cmd, args)
		fmt.Println("Starting processes...")
		runStart(cmd, args)
		return
	}
	fmt.Println("Restart complete")
}

func runLogs(cmd *cobra.Command, args []string) {
	follow := viper.GetBool("follow")
	lines := viper.GetInt("lines")

	// Try to detect running server and connect via API
	port, err := client.DetectServerPort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure guvnor server is running with: guvnor start\n")
		os.Exit(1)
	}

	apiClient := client.NewClient(port)

	processName := ""
	if len(args) > 0 {
		processName = args[0]
	}

	if processName != "" {
		fmt.Printf("Showing logs for app: %s (last %d lines)\n", processName, lines)
	} else {
		fmt.Printf("Showing logs for all apps (last %d lines)\n", lines)
	}

	// Get initial logs
	entries, err := apiClient.GetLogs(processName, lines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get logs: %v\n", err)
		os.Exit(1)
	}

	// Display logs
	for _, entry := range entries {
		fmt.Println(logs.FormatEntry(entry))
	}

	// If follow mode, stream new logs
	if follow {
		fmt.Printf("\n=== Following logs (Ctrl+C to stop) ===\n")
		
		err := apiClient.StreamLogs(processName, func(newEntries []logs.LogEntry) {
			for _, entry := range newEntries {
				fmt.Println(logs.FormatEntry(entry))
			}
		})
		
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error streaming logs: %v\n", err)
			os.Exit(1)
		}
	}
}



func runShell(cmd *cobra.Command, args []string) {
	fmt.Println("Guv'nor Interactive Shell")
	fmt.Println("Type 'help' for commands, 'quit' to exit")
	fmt.Println()

	// Simple interactive shell
	for {
		fmt.Print("guvnor> ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			continue
		}

		switch strings.TrimSpace(input) {
		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  status  - Show process status")
			fmt.Println("  start   - Start all processes")
			fmt.Println("  stop    - Stop all processes")
			fmt.Println("  restart - Restart all processes")
			fmt.Println("  logs    - Show recent logs")
			fmt.Println("  ps      - Show running processes")
			fmt.Println("  quit    - Exit shell")
		case "status":
			runStatus(cmd, args)
		case "start":
			runStart(cmd, args)
		case "stop":
			runStop(cmd, args)
		case "restart":
			runRestart(cmd, args)
		case "logs":
			runLogs(cmd, args)
		case "ps":
			// Show managed processes instead of shell command
			runStatus(cmd, args)
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return
		case "":
			continue
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", input)
		}
	}
}

func runValidate(cmd *cobra.Command, args []string) {
	fmt.Println("Validating configuration...")

	errors := 0
	warnings := 0

	// Validate Procfile
	if pf, err := loadProcfile(); err != nil {
		fmt.Printf("ERROR: Procfile validation failed: %v\n", err)
		errors++
	} else {
		fmt.Printf("OK: Procfile (%d processes)\n", len(pf.Processes))

		// Check environment warnings
		envWarnings := pf.ValidateEnvironment()
		for _, warning := range envWarnings {
			fmt.Printf("WARNING: %s\n", warning)
			warnings++
		}
	}

	// Validate config
	if _, err := loadConfig(); err != nil {
		fmt.Printf("ERROR: Configuration validation failed: %v\n", err)
		errors++
	} else {
		fmt.Println("OK: Configuration file")
	}

	// Validate environment
	if envConfig, err := env.LoadDotEnv("."); err != nil {
		fmt.Printf("WARNING: No .env files found\n")
		warnings++
	} else {
		fmt.Printf("OK: Environment (%d variables from %d files)\n",
			len(envConfig.Variables), len(envConfig.Files))
	}

	fmt.Printf("\nValidation complete: %d errors, %d warnings\n", errors, warnings)

	if errors > 0 {
		fmt.Println("Fix errors before running 'guvnor start'")
		os.Exit(1)
	} else if warnings > 0 {
		fmt.Println("Consider addressing warnings for production use")
	} else {
		fmt.Println("Configuration is valid!")
	}
}

func runStatus(cmd *cobra.Command, args []string) {
	var appName string
	if len(args) > 0 {
		appName = args[0]
		fmt.Printf("App Status: %s\n", appName)
	} else {
		fmt.Println("App Status (All):")
	}

	// Try to connect to running server via API
	port, err := client.DetectServerPort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure guvnor server is running with: guvnor start\n")
		os.Exit(1)
	}

	apiClient := client.NewClient(port)
	processInfo, err := apiClient.GetStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get status: %v\n", err)
		os.Exit(1)
	}
	
	// Filter by app name if specified
	if appName != "" {
		filtered := []process.ProcessInfo{}
		for _, info := range processInfo {
			if info.Name == appName {
				filtered = append(filtered, info)
				break
			}
		}
		if len(filtered) == 0 {
			fmt.Printf("App '%s' not found\n", appName)
			return
		}
		processInfo = filtered
	}

	if len(processInfo) > 0 {
		fmt.Printf("\n%-15s %-8s %-10s %-8s %-8s %-12s %s\n", 
			"APP", "PID", "STATUS", "RESTARTS", "PORT", "UPTIME", "COMMAND")
		fmt.Printf("%-15s %-8s %-10s %-8s %-8s %-12s %s\n", 
			"---", "---", "------", "--------", "----", "------", "-------")

		for _, info := range processInfo {
			pidStr := fmt.Sprintf("%d", info.PID)
			
			portStr := "-"
			if info.Port > 0 {
				portStr = fmt.Sprintf("%d", info.Port)
			}

			// Calculate uptime
			uptime := time.Since(info.StartTime).Truncate(time.Second)
			uptimeStr := formatDuration(uptime)

			// Build command string
			command := info.Command
			if len(info.Args) > 0 {
				command += " " + strings.Join(info.Args, " ")
			}
			if len(command) > 35 {
				command = command[:32] + "..."
			}

			// Color code status
			var statusDisplay string
			switch strings.ToLower(info.Status) {
			case "running":
				statusDisplay = "\033[32mrunning\033[0m"  // Green
			case "starting":
				statusDisplay = "\033[33mstarting\033[0m" // Yellow
			case "stopping":
				statusDisplay = "\033[33mstopping\033[0m" // Yellow
			case "failed":
				statusDisplay = "\033[31mfailed\033[0m"   // Red
			default:
				statusDisplay = info.Status
			}

			fmt.Printf("%-15s %-8s %-18s %-8d %-8s %-12s %s\n", 
				info.Name, pidStr, statusDisplay, info.Restarts, portStr, uptimeStr, command)
		}
	} else {
		// If no processes are running, show Procfile processes
		pf, err := loadProcfile()
		if err != nil {
			fmt.Printf("No running processes found and could not load Procfile: %v\n", err)
			return
		}

		fmt.Printf("\n%-15s %-8s %-18s %s\n", "PROCESS", "PID", "STATUS", "COMMAND")
		fmt.Printf("%-15s %-8s %-18s %s\n", "-------", "---", "------", "-------")

		for _, process := range pf.Processes {
			command := pf.SubstituteCommand(&process)
			if len(command) > 50 {
				command = command[:47] + "..."
			}
			fmt.Printf("%-15s %-8s %-18s %s\n", process.Name, "-", "\033[90mstopped\033[0m", command)
		}
	}
}

// Helper functions

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	} else {
		return fmt.Sprintf("%dd%dh", int(d.Hours()/24), int(d.Hours())%24)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func loadProcfile() (*procfile.Procfile, error) {
	procfilePath, err := procfile.FindProcfile(".")
	if err != nil {
		return nil, err
	}
	return procfile.ParseProcfile(procfilePath)
}

func loadConfig() (*config.Config, error) {
	configPath := "guvnor.yaml"
	if configFile != "" {
		configPath = configFile
	}
	return config.Load(configPath)
}

func createSmartConfig(apps []*discovery.App, minimal bool) *config.Config {
	cfg := &config.Config{
		Server: config.ServerConfig{
			HTTPPort:  8080,
			HTTPSPort: 8443,
			LogLevel:  "info",
		},
		TLS: config.TLSConfig{
			Enabled:  false,
			AutoCert: true,
		},
	}

	if !minimal {
		for _, app := range apps {
			appCfg := config.AppConfig{
				Name:    app.Name,
				Port:    app.Port,
				Command: app.Command,
				Args:    app.Args,
			}
			cfg.Apps = append(cfg.Apps, appCfg)
		}
	}

	return cfg
}

func updateGitignore(path string) error {
	entries := []string{
		"# Guv'nor",
		".env",
		".env.local",
		".env.*.local",
		"guvnor.log",
		"pids/",
		"logs/",
	}

	// Check if .gitignore exists
	var existing []string
	if content, err := os.ReadFile(path); err == nil {
		existing = strings.Split(string(content), "\n")
	}

	// Check which entries need to be added
	var toAdd []string
	for _, entry := range entries {
		found := false
		for _, line := range existing {
			if strings.TrimSpace(line) == entry {
				found = true
				break
			}
		}
		if !found {
			toAdd = append(toAdd, entry)
		}
	}

	// Append missing entries
	if len(toAdd) > 0 {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		if len(existing) > 0 && existing[len(existing)-1] != "" {
			file.WriteString("\n")
		}

		for _, entry := range toAdd {
			file.WriteString(entry + "\n")
		}
	}

	return nil
}

// Helper functions for command execution
func runCommand(cmd string) error {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	execCmd := exec.Command(parts[0], parts[1:]...)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

func runCommandOutput(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}

	execCmd := exec.Command(parts[0], parts[1:]...)
	output, err := execCmd.Output()
	if err != nil {
		return ""
	}

	return string(output)
}

// Certificate management commands

func runCertInfo(cmd *cobra.Command, args []string) {
	fmt.Println("Certificate Information:")
	
	// Load configuration to get certificate directory
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	
	if !cfg.TLS.Enabled {
		fmt.Println("TLS is not enabled in configuration")
		return
	}
	
	// Try to create certificate manager to get info
	certConfig := &cert.Config{
		Enabled:    cfg.TLS.Enabled,
		AutoCert:   cfg.TLS.AutoCert,
		CertDir:    cfg.TLS.CertDir,
		Email:      cfg.TLS.Email,
		Domains:    cfg.TLS.Domains,
		Staging:    cfg.TLS.Staging,
		ForceHTTPS: cfg.TLS.ForceHTTPS,
	}
	
	certMgr, err := cert.New(certConfig, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create certificate manager: %v\n", err)
		os.Exit(1)
	}
	
	certs, err := certMgr.GetCertificateInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get certificate info: %v\n", err)
		os.Exit(1)
	}
	
	if len(certs) == 0 {
		fmt.Println("No certificates found")
		return
	}
	
	fmt.Printf("%-30s %-12s %-20s %-20s %s\n", "DOMAIN", "STATUS", "NOT BEFORE", "NOT AFTER", "PATH")
	fmt.Printf("%-30s %-12s %-20s %-20s %s\n", "------", "------", "----------", "---------", "----")
	
	for _, cert := range certs {
		status := "valid"
		if cert.IsExpired {
			status = "expired"
		} else if time.Until(cert.NotAfter) < 30*24*time.Hour {
			status = "expiring"
		}
		
		fmt.Printf("%-30s %-12s %-20s %-20s %s\n",
			cert.Domain,
			status,
			cert.NotBefore.Format("2006-01-02 15:04"),
			cert.NotAfter.Format("2006-01-02 15:04"),
			cert.Path,
		)
	}
}

func runCertRenew(cmd *cobra.Command, args []string) {
	fmt.Println("Renewing certificates...")
	
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	
	if !cfg.TLS.Enabled {
		fmt.Println("TLS is not enabled in configuration")
		return
	}
	
	certConfig := &cert.Config{
		Enabled:    cfg.TLS.Enabled,
		AutoCert:   cfg.TLS.AutoCert,
		CertDir:    cfg.TLS.CertDir,
		Email:      cfg.TLS.Email,
		Domains:    cfg.TLS.Domains,
		Staging:    cfg.TLS.Staging,
		ForceHTTPS: cfg.TLS.ForceHTTPS,
	}
	
	certMgr, err := cert.New(certConfig, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create certificate manager: %v\n", err)
		os.Exit(1)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	if err := certMgr.RenewCertificates(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to renew certificates: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Certificate renewal completed")
}

func runCertCleanup(cmd *cobra.Command, args []string) {
	fmt.Println("Cleaning up certificates...")
	
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	
	if !cfg.TLS.Enabled {
		fmt.Println("TLS is not enabled in configuration")
		return
	}
	
	certConfig := &cert.Config{
		Enabled:    cfg.TLS.Enabled,
		AutoCert:   cfg.TLS.AutoCert,
		CertDir:    cfg.TLS.CertDir,
		Email:      cfg.TLS.Email,
		Domains:    cfg.TLS.Domains,
		Staging:    cfg.TLS.Staging,
		ForceHTTPS: cfg.TLS.ForceHTTPS,
	}
	
	certMgr, err := cert.New(certConfig, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create certificate manager: %v\n", err)
		os.Exit(1)
	}
	
	if err := certMgr.Cleanup(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to cleanup certificates: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Certificate cleanup completed")
}
