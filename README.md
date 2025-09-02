# Guvnor

Simple, fast web application deployment and process management.

**Guvnor** replaces complex infrastructure with a single binary. Zero configuration required - just works!

## Quick Start

```bash
# Initialize in your project directory
guvnor init

# Start your applications
guvnor start

# Production with TLS
guvnor start --domain myapp.com --email admin@myapp.com
```

## Key Features

- **Zero Configuration** - Auto-detects applications and frameworks
- **Automatic TLS** - Let's Encrypt certificates with zero setup  
- **Multi-App Support** - Virtual host routing with per-app configuration
- **Single Binary** - No dependencies, easy deployment
- **Process Management** - Automatic restarts and health monitoring

## Basic Commands

```bash
# Setup and management
guvnor init                 # Generate configuration
guvnor start [app-name]     # Start apps
guvnor stop [app-name]      # Stop apps  
guvnor status [app-name]    # Check status
guvnor logs [app-name]      # View logs
guvnor validate             # Check configuration
```

## Configuration Example

```yaml
# guvnor.yaml
server:
  http_port: 8080
  https_port: 8443

apps:
  - name: web-app
    hostname: web.localhost
    port: 3000
    command: node
    args: ["server.js"]
    
  - name: api-service  
    hostname: api.localhost
    port: 8000
    command: uvicorn
    args: ["main:app", "--host", "0.0.0.0", "--port", "8000"]
    tls:
      enabled: true
      auto_cert: true
      email: admin@example.com
```


## Installation

```bash
# Download latest release
curl -sSL https://github.com/gleicon/guvnor/releases/latest/download/guvnor-$(uname -s)-$(uname -m) -o guvnor
chmod +x guvnor
sudo mv guvnor /usr/local/bin/

# Or install with Go
go install github.com/gleicon/guvnor/cmd/guvnor@latest
```

## Documentation

See the [docs/](docs/) directory for comprehensive guides:

- [Platform Guides](docs/) - Next.js, React, Go, Rust, PHP, Java
- [SystemD Service](docs/systemd.md) - Running as system service
- [Configuration](docs/configuration.md) - Advanced configuration options
- [Examples](docs/examples.md) - Real-world usage examples

## License

MIT License - see [LICENSE](LICENSE) file
