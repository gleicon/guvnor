# Guv'nor

> Simple, fast web application deployment and process management.

**Guv'nor** replaces a bunch moving parts required to run your application in dev or production environment with a single binary. Zero configuration required - just works!

## Key Features

**Zero Configuration** - Auto-detects Python, Node.js, Go, Rust apps
**Automatic TLS** - Let's Encrypt certificates with zero setup
**Lightning Fast** - Sub-second startup, optimized for speed
**Smart Defaults** - Works perfectly out of the box
**Single Binary** - No dependencies, easy deployment
**Hot Reload** - Automatic restarts and health monitoring

## Quick Start

### Zero Config Deployment
```bash
# In any directory with apps (Python, Node.js, Go, Rust)
guvnor init

# Output:
# Initializing Guv'nor in: .
# Detecting applications...
# Found 2 applications:
#   - my-api (python)
#   - my-web (nodejs)
# Created: Procfile
# Created: .env
# Created: guvnor.yaml

# Start everything
guvnor start
```

### Production Deployment
```bash
# Deploy with automatic TLS
guvnor start --domain myapp.com --email admin@myapp.com

# Automatic:
# - Let's Encrypt certificates
# - Domain routing
# - Health monitoring
# - Process management
```

## Smart Detection

Guv'nor automatically detects and optimally configures:

### Python Applications
- **Django**: Auto-detects `manage.py`, configures `runserver`
- **Flask**: Finds Flask app, sets `FLASK_APP` environment
- **FastAPI**: Detects FastAPI imports, configures `uvicorn`
- **Streamlit**: Finds streamlit apps, configures ports

### Node.js Applications
- **Express**: Auto-detects Express dependencies
- **Next.js**: Configures dev/production modes
- **React**: Sets up development server
- **Generic**: Uses `package.json` scripts smartly

### Other Languages
- **Go**: Detects `go.mod`, runs with `go run .`
- **Rust**: Finds `Cargo.toml`, uses `cargo run`
- **Docker**: Detects `Dockerfile`, manages containers

## Usage Examples

### Basic Commands
```bash
# Initialize and start applications
guvnor init                      # Setup configuration
guvnor start                     # Start all apps
guvnor start web-app             # Start only specific app
guvnor start --daemon            # Run as background daemon

# Management
guvnor status                    # Show all apps
guvnor status web-app            # Show specific app status
guvnor stop                      # Stop all apps
guvnor stop api-service          # Stop specific app
guvnor restart web-app           # Restart specific app
guvnor logs                      # View all app logs
guvnor logs web-app              # View specific app logs
guvnor logs -f api-service       # Follow specific app logs
guvnor shell                     # Interactive management
guvnor validate                  # Check configuration
```

### Smart Configuration Generation

```bash
# Auto-generate optimized config
guvnor init

# Generated config includes:
# - Detected app types and frameworks
# - Optimal ports and health checks
# - Development-friendly defaults
# - Production deployment settings
```

## Configuration

### Multi-App Configuration (guvnor.yaml)
```yaml
# Multiple apps with different hostnames
server:
  http_port: 8080      # Non-privileged for development
  https_port: 8443     # Non-privileged for development
  log_level: info

apps:
  - name: web-frontend
    hostname: web.localhost    # Virtual host routing
    port: 3000                 # Auto-assigned if not specified
    command: node
    args: ["server.js"]
    
    # Per-app TLS settings
    tls:
      enabled: false           # HTTP only for this app

  - name: api-backend
    hostname: api.localhost    # Different hostname
    port: 8000                 # Framework-appropriate port
    command: uvicorn           # Auto-detected FastAPI
    args: ["main:app", "--host", "0.0.0.0", "--port", "8000"]
    
    # TLS enabled for API
    tls:
      enabled: true
      auto_cert: true
      email: api@localhost
      staging: true
    
    health_check:
      enabled: true
      path: /docs             # FastAPI-specific health check
      interval: 30s

  - name: admin-panel
    # hostname auto-generated as "admin-panel.localhost"
    # port auto-assigned (6000 in this case)
    command: python
    args: ["-m", "streamlit", "run", "admin.py"]
    
    tls:
      enabled: true
      auto_cert: true
      email: admin@localhost

# Global TLS settings (fallback)
tls:
  enabled: true              # Default for production
  cert_dir: ./certs
  force_https: false         # Allow HTTP for development
```

### Production Configuration
```yaml
# Production multi-app setup
server:
  http_port: 80              # Standard HTTP
  https_port: 443           # Standard HTTPS
  log_level: warn           # Production logging

apps:
  - name: web-app
    hostname: myapp.com      # Main domain
    port: 3000
    command: node
    args: ["dist/server.js"]
    
    tls:
      enabled: true
      auto_cert: true
      email: admin@myapp.com
      staging: false         # Production certificates

  - name: api-service
    hostname: api.myapp.com  # API subdomain
    port: 8000
    command: uvicorn
    args: ["main:app", "--host", "0.0.0.0", "--port", "8000"]
    
    tls:
      enabled: true
      auto_cert: true
      email: api@myapp.com
      staging: false

# Global TLS settings
tls:
  enabled: true             # TLS enabled globally
  cert_dir: /var/lib/guvnor/certs
  force_https: true         # Redirect HTTP->HTTPS
```

## Procfile Support

Guv'nor is fully compatible with Heroku/Foreman Procfiles:

```procfile
# Procfile - Generated by Guv'nor
web: gunicorn app.wsgi:application --bind 0.0.0.0:$PORT
worker: celery -A app worker --loglevel=info
beat: celery -A app beat --loglevel=info
redis: redis-server --port 6379
```


## Compare Guv'nor with similar tools:

### vs. Docker + docker-compose
```bash
# Old way (Docker)
$ docker-compose up -d          # Slow startup
$ docker-compose logs -f        # Complex logging
$ vim docker-compose.yml        # Manual configuration

# New way (Guv'nor)
$ guvnor init                   # Auto-detect everything
$ guvnor start                  # Instant startup
```

### vs. nginx + systemd
```bash
# Old way (nginx + systemd)
$ sudo vim /etc/nginx/sites-available/myapp  # Manual config
$ sudo systemctl reload nginx                # Manual reload
$ sudo systemctl enable myapp               # Manual service
$ sudo certbot --nginx                      # Manual TLS

# New way (Guv'nor)
$ guvnor start --domain myapp.com --email admin@myapp.com  # Everything automatic
```

### vs. Kubernetes
```bash
# Old way (Kubernetes)
$ kubectl apply -f deployment.yaml    # Complex YAML
$ kubectl apply -f service.yaml       # More YAML
$ kubectl apply -f ingress.yaml       # Even more YAML
$ kubectl get pods                    # Check status

# New way (Guv'nor)
$ guvnor init                         # Auto-detect
$ guvnor start                        # Just works
```

## Multi-App Management

Guvnor supports running multiple applications with different configurations in a single instance:

### Virtual Host Routing
Each app gets its own hostname for request routing:
```bash
# Requests to different hostnames route to different apps
curl http://web.localhost:8080        # Routes to web-frontend (port 3000)
curl http://api.localhost:8080        # Routes to api-backend (port 8000)  
curl http://admin-panel.localhost:8080 # Routes to admin-panel (port 6000)
```

### Per-App TLS Configuration
Each app can have different TLS settings:
```yaml
apps:
  - name: public-web
    hostname: myapp.com
    tls:
      enabled: true      # HTTPS with Let's Encrypt
      
  - name: internal-api
    hostname: internal.myapp.com
    tls:
      enabled: false     # HTTP only (internal use)
```

### App-Specific Operations
Manage individual apps without affecting others:
```bash
guvnor start api-backend     # Start only API
guvnor stop web-frontend     # Stop only frontend
guvnor logs admin-panel      # View only admin logs
guvnor restart api-backend   # Restart only API
```

### Smart Defaults
- Hostnames auto-generated as `{app-name}.localhost`
- Ports auto-assigned (3000, 4000, 5000, etc.)
- TLS email inherits from global settings
- Health checks adapt to app type

## Advanced Features

### Environment-Specific Deployment
```bash
# Development (automatic)
guvnor start --dev

# Production deployment
guvnor start --domain myapp.com --email admin@myapp.com

# Staging with test certificates
guvnor start --domain staging.myapp.com --email admin@myapp.com --staging
```

### Health Monitoring
```bash
# Built-in health monitoring
guvnor status

# Output:
# App Status (All):
# APP             PID      STATUS     RESTARTS PORT     UPTIME       COMMAND
# ---             ---      ------     -------- ----     ------       -------
# web-frontend    1234     running    0        3000     2h 45m       node server.js
# api-backend     1235     running    0        8000     2h 45m       uvicorn main:app
# admin-panel     1236     running    1        6000     1h 30m       python -m streamlit run admin.py

# App-specific status
guvnor status web-frontend

# Output:
# App Status: web-frontend
# APP             PID      STATUS     RESTARTS PORT     UPTIME       COMMAND
# ---             ---      ------     -------- ----     ------       -------
# web-frontend    1234     running    0        3000     2h 45m       node server.js
```

### Process Management
- **Automatic Restarts**: Failed processes restart automatically
- **Health Checks**: HTTP/TCP/Command-based health monitoring
- **Graceful Shutdown**: SIGTERM -> wait -> SIGKILL sequence
- **Resource Limits**: Memory and CPU constraints
- **Log Aggregation**: Centralized logging with rotation

## Installation
### From github releases

Check [the release page](https://github.com/gleicon/guvnor/releases) for your platform and download the binary.

Create a short link:
`sudo ln -sf /usr/local/bin/guvnor /usr/local/bin/gv`

### Download binary directly with curl
```bash
# macOS/Linux
curl -sSL https://github.com/gleicon/guvnor/releases/latest/download/guvnor-$(uname -s)-$(uname -m) -o guvnor
chmod +x guvnor
sudo mv guvnor /usr/local/bin/

# Create short alias
sudo ln -sf /usr/local/bin/guvnor /usr/local/bin/gv

# Or with Go
go install github.com/gleicon/guvnor/cmd/guvnor@latest
```

### Build from Source
```bash
git clone https://github.com/gleicon/guvnor.git
cd guvnor
make build

# This creates both guvnor and gv binaries
```

## Architecture

Guv'nor consists of:

- **Discovery Engine**: Auto-detects applications and frameworks
- **Process Manager**: Supervises application lifecycles
- **Reverse Proxy**: Routes requests with TLS termination
- **Certificate Manager**: Automatic Let's Encrypt integration
- **Health Monitor**: Continuous application monitoring
- **Configuration Engine**: Smart defaults with override capability

## Comparison Matrix

| Feature | Guv'nor | nginx+systemd | Docker | Kubernetes |
|---------|---------|---------------|--------|------------|
| Setup Time | 0 seconds | 30+ minutes | 10+ minutes | Hours+ |
| Configuration | Zero | Complex | Moderate | Very Complex |
| TLS Setup | Automatic | Manual | Manual | Complex |
| Resource Usage | Minimal | Low | Moderate | High |
| Learning Curve | None | Moderate | Moderate | Steep |
| Production Ready | Yes | Yes | Yes | Yes |
| 12-Factor Apps | Native | Manual | Manual | Native |
| Single Binary | Yes | No | No | No |

## 12-Factor Application Support

Guv'nor has native support for 12-factor applications:

- **Procfile**: Native Procfile support (Heroku/Foreman compatible)
- **Environment Variables**: `.env` file support with variable substitution
- **Process Management**: Native process supervision and scaling
- **Logging**: Centralized log aggregation
- **Port Binding**: Automatic port assignment and routing
- **Disposability**: Graceful startup and shutdown

## License

MIT License - see [LICENSE](LICENSE) file

## Support

- **Documentation**: [EXAMPLES.md](EXAMPLES.md) for detailed examples
- **Community**: [GitHub Discussions](https://github.com/gleicon/guvnor/discussions)
- **Issues**: [GitHub Issues](https://github.com/gleicon/guvnor/issues)
- **Feature Requests**: [GitHub Issues](https://github.com/gleicon/guvnor/issues)

## Motivation: Why Guv'nor?

Years of application development and deployment have taught us that simplicity is key. Guv'nor is designed to be a single binary that replaces the complexity of nginx+systemd+docker-compose with a simple, fast, and easy-to-use deployment and process management solution.

My stack revolved around parts of [foreman](https://github.com/ddollar/foreman), [supervisord](https://supervisord.org/), [rc.S](https://en.wikipedia.org/wiki/Init) style scripts, [12Factor](https://12factor.net/), my ancient [nginx](https://nginx.org/) reverse proxy config and [let's encrypt](https://letsencrypt.org/) facilities.

Tools like `uv` and `pm2` as well as `supervisord` and the evolution of package management inspired me and made it logical to glue a simple process manager that could support a reverse proxy like the times of inetd.

I run software locally, on cheap VPS, VMs, or Containers on k8s. I want to run my apps, have TLS, and be able to manage them easily.

Guv'nor simplifies this process by providing a single binary that handles all the necessary configurations and dependencies.

The name is inspired by the great MFDoom and Jneiro Jarel song and video (https://www.youtube.com/watch?v=WW-9TcDTKa8) !

---

<p align="center">
<strong>Guv'nor: Simple, fast, reliable process management</strong><br>
<em>Zero configuration. Lightning fast. Just works.</em>
</p>
