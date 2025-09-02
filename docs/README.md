# Guvnor Documentation

Comprehensive documentation for Guvnor - simple, fast web application deployment and process management.

## Getting Started

- [Installation](#installation) - Download and install Guvnor
- [Quick Start](#quick-start) - Get running in minutes
- [Configuration](configuration.md) - Complete configuration reference
- [Examples](examples.md) - Real-world usage examples

## Platform Guides

Framework-specific setup guides:

- [Next.js Applications](nextjs.md) - React framework with SSR/SSG
- [React Applications](react.md) - Single-page React applications  
- [Go Applications](go.md) - Go web services and APIs
- [Rust Applications](rust.md) - Rust web applications
- [PHP Applications](php.md) - PHP web applications and frameworks
- [Java Applications](java.md) - Java web applications and Spring Boot

## Production Deployment

- [SystemD Service](systemd.md) - Running as system service
- [Multi-App Setup](configuration.md#multi-app-configuration) - Multiple applications with virtual hosts
- [TLS Configuration](configuration.md#tls-configuration) - HTTPS and Let's Encrypt setup

## Installation

```bash
# Download latest release
curl -sSL https://github.com/gleicon/guvnor/releases/latest/download/guvnor-$(uname -s)-$(uname -m) -o guvnor
chmod +x guvnor
sudo mv guvnor /usr/local/bin/

# Or install with Go
go install github.com/gleicon/guvnor/cmd/guvnor@latest
```

## Quick Start

```bash
# Initialize in your project directory
guvnor init

# Start your applications
guvnor start

# Production with TLS
guvnor start --domain myapp.com --email admin@myapp.com
```

## Key Concepts

- **Zero Configuration**: Auto-detects applications and generates optimal configuration
- **Multi-App Support**: Run multiple applications with virtual host routing
- **Process Management**: Automatic restarts, health monitoring, and graceful shutdowns
- **TLS Termination**: Automatic Let's Encrypt certificates with per-app configuration
- **Single Binary**: No dependencies, easy deployment

## Basic Commands

```bash
guvnor init                 # Generate configuration
guvnor start [app-name]     # Start apps
guvnor stop [app-name]      # Stop apps  
guvnor status [app-name]    # Check status
guvnor logs [app-name]      # View logs
guvnor validate             # Check configuration
```

## Architecture

Guvnor provides:

1. **Application Detection**: Automatically detects and configures applications
2. **Process Supervision**: Manages application lifecycles with health monitoring
3. **Reverse Proxy**: Routes requests with hostname-based virtual hosts
4. **TLS Management**: Automatic certificate provisioning and renewal
5. **Log Aggregation**: Centralized logging with circular buffer storage