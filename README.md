# Guvnor

Process manager and reverse proxy in one binary.

## Quick Start

```bash
cd your-project
guvnor init
guvnor start
```

Visit `http://your-project.localhost:8080`

## Features

- Auto-detects Node.js, Python, Go, Rust, PHP, Java
- Process management with health checks
- Virtual host routing
- Automatic HTTPS via Let's Encrypt
- Zero dependencies

## Commands

```bash
guvnor init                 # Generate config
guvnor start [app]          # Start apps
guvnor stop [app]           # Stop apps
guvnor status [app]         # Show status
guvnor logs [app]           # View logs
```

## Config

```yaml
apps:
  - name: web
    hostname: web.localhost
    command: node
    args: ["server.js"]
    
  - name: api
    hostname: api.localhost
    command: uvicorn
    args: ["main:app"]
    tls:
      enabled: true
      auto_cert: true
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

## How It Works

Request → Guvnor (reverse proxy) → Your app

1. Detects apps in your project
2. Generates config
3. Starts processes
4. Routes by hostname

Config priority: `guvnor.yaml` > `Procfile` > auto-detect

## Docs

- [Getting Started](docs/getting-started.md)
- [Workflows](docs/workflows.md)
- [Config Reference](docs/configuration.md)

## Documentation

- [Getting Started](docs/getting-started.md) - Step-by-step scenarios
- [Common Workflows](docs/workflows.md) - Daily development tasks
- [Configuration](docs/configuration.md) - All configuration options  
- [Platform Guides](docs/) - Next.js, React, Go, Rust, PHP, Java
- [Examples](docs/examples.md) - Real-world configurations
- [Production Setup](docs/systemd.md) - Running as system service
- [Architecture](docs/architecture.md) - How Guvnor works internally

## License

MIT License - see [LICENSE](LICENSE) file
