# Getting Started

Get up and running with Guvnor in minutes.

## Installation

<div class="command-section">
<div class="command-title">Download Binary</div>
<div class="command-description">Install the latest release directly</div>

```bash
curl -sSL https://github.com/gleicon/guvnor/releases/latest/download/guvnor-$(uname -s)-$(uname -m) -o guvnor
chmod +x guvnor
sudo mv guvnor /usr/local/bin/
```

</div>

<div class="command-section">
<div class="command-title">Install with Go</div>
<div class="command-description">Build from source using Go toolchain</div>

```bash
go install github.com/gleicon/guvnor/cmd/guvnor@latest
```

</div>

## Quick Start Scenarios

### New Project

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>Create a new project</strong>
</div>
<div class="config-content">

```bash
mkdir my-app && cd my-app
echo 'console.log("Hello")' > server.js
echo '{"name": "my-app"}' > package.json

guvnor init
guvnor start
```

</div>
</div>

<div>

**What happens:**
1. Guvnor detects Node.js from `package.json`
2. Generates `guvnor.yaml` with smart defaults
3. Starts your app at `http://my-app.localhost:8080`

**Generated config:**
```yaml
apps:
  - name: my-app
    hostname: my-app.localhost
    command: node
    args: ["server.js"]
```

</div>
</div>

### Existing Project

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>Add Guvnor to existing project</strong>
</div>
<div class="config-content">

```bash
cd my-existing-project
guvnor init
guvnor start
```

</div>
</div>

<div>

Guvnor auto-detects your project type:

| Project Type | Detection | Generated Command |
|--------------|-----------|-------------------|
| **Node.js** | `package.json` | `npm start` or `node server.js` |
| **Python** | `requirements.txt` | `python app.py` |
| **Go** | `go.mod` | `go run .` |
| **Rust** | `Cargo.toml` | `cargo run` |
| **PHP** | `composer.json` | `php -S 0.0.0.0:3000` |

</div>
</div>

### Heroku/Procfile App

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>Migrate from Heroku</strong>
</div>
<div class="config-content">

```bash
# Your existing Procfile:
# web: gunicorn app:app --port $PORT
# worker: celery -A app worker

guvnor init     # Reads Procfile
guvnor start    # Runs both processes
```

</div>
</div>

<div>

**Process mapping:**
- `web` → Gets hostname routing at `http://web.localhost:8080`
- `worker` → Runs in background (no HTTP routing)
- `.env` files are preserved and loaded automatically

</div>
</div>

## Multi-App Setup

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>Run multiple applications</strong>
</div>
<div class="config-content">

```yaml
# guvnor.yaml
apps:
  - name: frontend
    hostname: web.localhost
    command: npm
    args: ["start"]
    
  - name: api
    hostname: api.localhost
    command: uvicorn
    args: ["main:app"]
    
  - name: worker
    command: python
    args: ["worker.py"]
    # No hostname = background service
```

</div>
</div>

<div>

**Access your apps:**
- Frontend: `http://web.localhost:8080`
- API: `http://api.localhost:8080`
- Worker: Background process (no HTTP access)

Each app gets automatic:
- Process management
- Health monitoring  
- Log aggregation
- Restart on failure

</div>
</div>

## Production Deployment

<div class="api-section">

<span class="api-method">HTTPS</span> <code class="api-endpoint">automatic certificates</code>

Deploy with zero-config HTTPS using Let's Encrypt.

</div>

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>Production with auto HTTPS</strong>
</div>
<div class="config-content">

```bash
guvnor start --domain myapp.com --email admin@myapp.com
```

</div>
</div>

<div>

**Automatic setup:**
- Let's Encrypt SSL certificates
- HTTP → HTTPS redirects
- Production logging
- Process monitoring

**Configuration generated:**
```yaml
server:
  http_port: 80
  https_port: 443
  
apps:
  - name: myapp
    hostname: myapp.com
    tls:
      enabled: true
      auto_cert: true
      email: admin@myapp.com
```

</div>
</div>

## Daily Commands

<div class="command-section">
<div class="command-title">Process Management</div>
<div class="command-description">Common development tasks</div>

```bash
guvnor start           # Start all apps
guvnor start webapp    # Start specific app
guvnor stop            # Stop all apps  
guvnor restart api     # Restart specific app
guvnor status          # Show app status
guvnor logs            # View all logs
guvnor logs webapp -f  # Follow specific app logs
```

</div>

## Configuration Priority

How Guvnor decides what to run:

<div class="config-section">
<div>

### 1. guvnor.yaml (Primary)
Explicit configuration file created by `guvnor init` or manually.

### 2. Procfile (Fallback)  
Heroku-style process definitions for 12-factor app compatibility.

### 3. Auto-detection (Last Resort)
Scans project files to infer application type and commands.

</div>

<div class="config-example">
<div class="config-header">
<strong>Environment Variables</strong>
</div>
<div class="config-content">

**Priority (highest to lowest):**
1. `guvnor.yaml` app environment
2. `.env` file variables
3. System environment

</div>
</div>
</div>

## File Structure

After running `guvnor init`, you'll have:

```
my-project/
├── guvnor.yaml    # Primary configuration
├── .env           # Environment variables (optional)
├── Procfile       # Process definitions (optional)
└── your-app-files/
```

## Next Steps

<div class="config-section">
<div>

### Learn More
- [Configuration Reference](configuration.html) - Complete guvnor.yaml options
- [Common Workflows](workflows.html) - Daily development tasks
- [Examples](examples.html) - Real-world configurations

### Platform-Specific  
- [Next.js Guide](nextjs.html)
- [React Guide](react.html)
- [Go Guide](go.html)
- [Python Guide](python.html)

### Production
- [SystemD Service](systemd.html) - Run as system service
- [Architecture](architecture.html) - How Guvnor works

</div>

<div class="command-section">
<div class="command-title">Get Help</div>
<div class="command-description">Resources for troubleshooting</div>

- **Validate config:** `guvnor validate`
- **Debug logs:** `guvnor logs --level debug`
- **GitHub Issues:** [Report problems](https://github.com/gleicon/guvnor/issues)

</div>
</div>