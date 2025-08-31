package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// App represents a discovered application
type App struct {
	Name        string            `json:"name" yaml:"name"`
	Type        string            `json:"type" yaml:"type"`
	Path        string            `json:"path" yaml:"path"`
	Port        int               `json:"port" yaml:"port"`
	Command     string            `json:"command" yaml:"command"`
	Args        []string          `json:"args" yaml:"args"`
	Env         map[string]string `json:"env" yaml:"env"`
	HealthCheck string            `json:"health_check" yaml:"health_check"`
	Domain      string            `json:"domain,omitempty" yaml:"domain,omitempty"`
}

// DiscoverApps automatically detects applications in the given directory
// This is the core uv-inspired "just works" functionality
func DiscoverApps(dir string) ([]*App, error) {
	var apps []*App
	
	// Normalize directory path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}
	
	// Walk the directory to find application indicators
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip hidden directories and common ignore patterns
		if info.IsDir() && shouldSkipDir(info.Name()) {
			return filepath.SkipDir
		}
		
		if !info.IsDir() {
			if app := detectAppFromFile(path, absDir); app != nil {
				apps = append(apps, app)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}
	
	// Assign smart default ports
	assignPorts(apps)
	
	// Configure smart health checks
	configureHealthChecks(apps)
	
	return apps, nil
}

// detectAppFromFile detects application type from specific files
func detectAppFromFile(filePath, baseDir string) *App {
	filename := filepath.Base(filePath)
	dirPath := filepath.Dir(filePath)
	
	switch filename {
	case "requirements.txt":
		return detectPythonApp(dirPath, baseDir, "requirements")
	case "pyproject.toml":
		return detectPythonApp(dirPath, baseDir, "pyproject")
	case "Pipfile":
		return detectPythonApp(dirPath, baseDir, "pipenv")
	case "setup.py":
		return detectPythonApp(dirPath, baseDir, "setup")
	case "package.json":
		return detectNodeApp(filePath, dirPath, baseDir)
	case "go.mod":
		return detectGoApp(filePath, dirPath, baseDir)
	case "Cargo.toml":
		return detectRustApp(filePath, dirPath, baseDir)
	case "Dockerfile":
		return detectDockerApp(dirPath, baseDir)
	}
	
	return nil
}

// detectPythonApp detects Python applications with smart framework detection
func detectPythonApp(appDir, baseDir, detectionMethod string) *App {
	relPath, _ := filepath.Rel(baseDir, appDir)
	if relPath == "." {
		relPath = ""
	}
	
	appName := filepath.Base(appDir)
	if appName == "." {
		appName = filepath.Base(baseDir)
	}
	
	app := &App{
		Name: appName,
		Type: "python",
		Path: appDir,
		Env:  make(map[string]string),
	}
	
	// Smart framework detection
	framework := detectPythonFramework(appDir)
	
	switch framework {
	case "django":
		app.Command = "python"
		app.Args = []string{"manage.py", "runserver", "0.0.0.0:$PORT"}
		app.HealthCheck = "/admin/"
	case "flask":
		app.Command = "flask"
		app.Args = []string{"run", "--host=0.0.0.0", "--port=$PORT"}
		app.Env["FLASK_APP"] = findFlaskApp(appDir)
		app.HealthCheck = "/"
	case "fastapi":
		app.Command = "uvicorn"
		mainFile := findFastAPIMain(appDir)
		app.Args = []string{mainFile, "--host", "0.0.0.0", "--port", "$PORT"}
		app.HealthCheck = "/docs"
	case "streamlit":
		app.Command = "streamlit"
		mainFile := findStreamlitMain(appDir)
		app.Args = []string{"run", mainFile, "--server.port", "$PORT", "--server.address", "0.0.0.0"}
		app.HealthCheck = "/"
	default:
		// Generic Python app
		if mainFile := findPythonMain(appDir); mainFile != "" {
			app.Command = "python"
			app.Args = []string{mainFile}
		} else {
			app.Command = "python"
			app.Args = []string{"-m", "http.server", "$PORT"}
		}
		app.HealthCheck = "/"
	}
	
	return app
}

// detectNodeApp detects Node.js applications with smart framework detection
func detectNodeApp(packagePath, appDir, baseDir string) *App {
	relPath, _ := filepath.Rel(baseDir, appDir)
	if relPath == "." {
		relPath = ""
	}
	
	appName := filepath.Base(appDir)
	if appName == "." {
		appName = filepath.Base(baseDir)
	}
	
	app := &App{
		Name: appName,
		Type: "nodejs",
		Path: appDir,
		Env:  make(map[string]string),
	}
	
	// Parse package.json for smart detection
	packageData := parsePackageJSON(packagePath)
	
	// Use package.json name if available
	if packageData.Name != "" {
		app.Name = packageData.Name
	}
	
	// Smart script detection
	if script := packageData.Scripts["start"]; script != "" {
		parts := strings.Fields(script)
		if len(parts) > 0 {
			app.Command = parts[0]
			app.Args = parts[1:]
		}
	} else if script := packageData.Scripts["dev"]; script != "" {
		parts := strings.Fields(script)
		if len(parts) > 0 {
			app.Command = parts[0]
			app.Args = parts[1:]
		}
	} else {
		// Smart framework detection
		framework := detectNodeFramework(packageData)
		
		switch framework {
		case "next":
			app.Command = "npm"
			app.Args = []string{"run", "dev"}
			app.Env["PORT"] = "$PORT"
			app.HealthCheck = "/"
		case "express":
			if packageData.Main != "" {
				app.Command = "node"
				app.Args = []string{packageData.Main}
			} else {
				app.Command = "npm"
				app.Args = []string{"start"}
			}
			app.Env["PORT"] = "$PORT"
			app.HealthCheck = "/"
		case "react":
			app.Command = "npm"
			app.Args = []string{"start"}
			app.Env["PORT"] = "$PORT"
			app.HealthCheck = "/"
		default:
			// Generic Node.js app
			if packageData.Main != "" {
				app.Command = "node"
				app.Args = []string{packageData.Main}
			} else {
				app.Command = "npm"
				app.Args = []string{"start"}
			}
			app.Env["PORT"] = "$PORT"
		}
	}
	
	app.HealthCheck = "/"
	return app
}

// detectGoApp detects Go applications
func detectGoApp(goModPath, appDir, baseDir string) *App {
	relPath, _ := filepath.Rel(baseDir, appDir)
	if relPath == "." {
		relPath = ""
	}
	
	// Parse go.mod for module name
	moduleName := parseGoMod(goModPath)
	appName := filepath.Base(moduleName)
	if appName == "" {
		appName = filepath.Base(appDir)
	}
	
	app := &App{
		Name:        appName,
		Type:        "go",
		Path:        appDir,
		Command:     "go",
		Args:        []string{"run", "."},
		Env:         map[string]string{"PORT": "$PORT"},
		HealthCheck: "/",
	}
	
	return app
}

// detectRustApp detects Rust applications
func detectRustApp(cargoPath, appDir, baseDir string) *App {
	relPath, _ := filepath.Rel(baseDir, appDir)
	if relPath == "." {
		relPath = ""
	}
	
	appName := filepath.Base(appDir)
	
	app := &App{
		Name:        appName,
		Type:        "rust",
		Path:        appDir,
		Command:     "cargo",
		Args:        []string{"run"},
		Env:         map[string]string{"PORT": "$PORT"},
		HealthCheck: "/",
	}
	
	return app
}

// detectDockerApp detects Dockerized applications
func detectDockerApp(appDir, baseDir string) *App {
	relPath, _ := filepath.Rel(baseDir, appDir)
	if relPath == "." {
		relPath = ""
	}
	
	appName := filepath.Base(appDir)
	
	app := &App{
		Name:        appName,
		Type:        "docker",
		Path:        appDir,
		Command:     "docker",
		Args:        []string{"run", "--rm", "-p", "$PORT:$PORT", "."},
		Env:         map[string]string{"PORT": "$PORT"},
		HealthCheck: "/",
	}
	
	return app
}

// Helper functions for smart framework detection

func detectPythonFramework(appDir string) string {
	// Check for Django
	if fileExists(filepath.Join(appDir, "manage.py")) {
		return "django"
	}
	
	// Check requirements.txt for framework hints
	reqFile := filepath.Join(appDir, "requirements.txt")
	if content, err := os.ReadFile(reqFile); err == nil {
		contentStr := strings.ToLower(string(content))
		if strings.Contains(contentStr, "django") {
			return "django"
		}
		if strings.Contains(contentStr, "fastapi") {
			return "fastapi"
		}
		if strings.Contains(contentStr, "flask") {
			return "flask"
		}
		if strings.Contains(contentStr, "streamlit") {
			return "streamlit"
		}
	}
	
	// Check for FastAPI files
	if findFastAPIMain(appDir) != "" {
		return "fastapi"
	}
	
	// Check for Flask files
	if findFlaskApp(appDir) != "" {
		return "flask"
	}
	
	return "generic"
}

func detectNodeFramework(pkg *PackageJSON) string {
	// Check dependencies for framework indicators
	deps := make(map[string]bool)
	for dep := range pkg.Dependencies {
		deps[dep] = true
	}
	for dep := range pkg.DevDependencies {
		deps[dep] = true
	}
	
	if deps["next"] {
		return "next"
	}
	if deps["express"] {
		return "express"
	}
	if deps["react"] && deps["react-scripts"] {
		return "react"
	}
	
	return "generic"
}

// Smart port assignment
func assignPorts(apps []*App) {
	usedPorts := make(map[int]bool)
	defaultPorts := map[string]int{
		"python":  8000,
		"nodejs":  3000,
		"go":      8080,
		"rust":    8080,
		"docker":  8080,
	}
	
	for _, app := range apps {
		if app.Port == 0 {
			basePort := defaultPorts[app.Type]
			if basePort == 0 {
				basePort = 8000
			}
			
			port := basePort
			for usedPorts[port] {
				port++
			}
			
			app.Port = port
			usedPorts[port] = true
		}
	}
}

// Smart health check configuration
func configureHealthChecks(apps []*App) {
	for _, app := range apps {
		if app.HealthCheck == "" {
			app.HealthCheck = "/"
		}
	}
}

// Utility functions

func shouldSkipDir(name string) bool {
	skipDirs := []string{
		"node_modules", ".git", ".venv", "venv", "__pycache__",
		".pytest_cache", "target", "dist", "build", ".next",
		".cache", "coverage", ".nyc_output",
	}
	
	for _, skip := range skipDirs {
		if name == skip {
			return true
		}
	}
	
	return strings.HasPrefix(name, ".")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// File finder functions

func findPythonMain(dir string) string {
	candidates := []string{"main.py", "app.py", "server.py", "run.py"}
	for _, candidate := range candidates {
		if fileExists(filepath.Join(dir, candidate)) {
			return candidate
		}
	}
	return ""
}

func findFlaskApp(dir string) string {
	candidates := []string{"app.py", "main.py", "server.py", "application.py"}
	for _, candidate := range candidates {
		path := filepath.Join(dir, candidate)
		if fileExists(path) {
			// Check if it's actually a Flask app
			if content, err := os.ReadFile(path); err == nil {
				if strings.Contains(strings.ToLower(string(content)), "flask") {
					return candidate
				}
			}
		}
	}
	return "app.py" // default
}

func findFastAPIMain(dir string) string {
	candidates := []string{"main.py", "app.py", "api.py", "server.py"}
	for _, candidate := range candidates {
		path := filepath.Join(dir, candidate)
		if fileExists(path) {
			if content, err := os.ReadFile(path); err == nil {
				if strings.Contains(strings.ToLower(string(content)), "fastapi") {
					return strings.TrimSuffix(candidate, ".py") + ":app"
				}
			}
		}
	}
	return "main:app" // default
}

func findStreamlitMain(dir string) string {
	candidates := []string{"app.py", "main.py", "streamlit_app.py"}
	for _, candidate := range candidates {
		if fileExists(filepath.Join(dir, candidate)) {
			return candidate
		}
	}
	return "app.py"
}

// Package.json parsing

type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Main            string            `json:"main"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func parsePackageJSON(path string) *PackageJSON {
	var pkg PackageJSON
	
	content, err := os.ReadFile(path)
	if err != nil {
		return &pkg
	}
	
	json.Unmarshal(content, &pkg)
	return &pkg
}

func parseGoMod(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module ")
		}
	}
	
	return ""
}