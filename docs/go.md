# How to run your Go app with Guvnor

## Setup

```bash
cd my-go-app
go mod tidy
go build -o myapp
guvnor init
```

## Configuration

```yaml
# guvnor.yaml
server:
  http_port: 8080
  https_port: 8443
  log_level: info

apps:
  - name: go-app
    hostname: localhost
    port: 3000
    command: ./myapp
    working_dir: /path/to/your/go-app
    environment:
      GO_ENV: "production"
      PORT: "3000"
    health_check:
      enabled: true
      path: /health
      interval: 30s
    restart_policy:
      enabled: true
      max_retries: 5
      backoff: 3s

tls:
  enabled: false
  auto_cert: false
  cert_dir: ./certs
```

## Usage

```bash
# Production deployment
go build -o myapp               # Build first
guvnor start                    # Access: http://localhost:8080/
guvnor logs                     # View logs
guvnor status                   # Check status
guvnor stop                     # Stop app

# Development
guvnor start                    # If using dev config below
```

## Alternative Configurations

### Development Mode
```yaml
apps:
  - name: go-dev
    command: go
    args: ["run", "main.go"]
    environment:
      GO_ENV: "development"
```

### Cross-compiled Binary
```yaml
apps:
  - name: go-app
    command: ./myapp-linux-amd64
    environment:
      GO_ENV: "production"
```

### With Air (Hot Reload)
```yaml
apps:
  - name: go-air
    command: air
    args: ["-c", ".air.toml"]
    environment:
      GO_ENV: "development"
```

## Build Commands

```bash
# Local build
go build -o myapp

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o myapp-linux-amd64

# Build with optimizations
go build -ldflags "-w -s" -o myapp

# Static binary
CGO_ENABLED=0 go build -a -ldflags "-w -s" -o myapp
```

## Required go.mod

```go
module my-go-app

go 1.21

require (
    // Your dependencies
)
```