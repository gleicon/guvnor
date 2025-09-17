# Getting Started

## Installation

Download binary:
```bash
curl -sSL https://github.com/gleicon/guvnor/releases/latest/download/guvnor-$(uname -s)-$(uname -m) -o guvnor
chmod +x guvnor
sudo mv guvnor /usr/local/bin/
```

Or install with Go:
```bash
go install github.com/gleicon/guvnor/cmd/guvnor@latest
```

## New Project

```bash
mkdir my-app && cd my-app
echo 'console.log("Hello")' > server.js
echo '{"name": "my-app"}' > package.json

guvnor init
guvnor start
```

Visit `http://my-app.localhost:8080`

## Existing Project

```bash
cd my-project
guvnor init
guvnor start
```

## Foreman and Procfile/Heroku style app

```bash
# Existing Procfile:
# web: gunicorn app:app
# worker: celery -A app worker

guvnor init     # Uses Procfile
guvnor start
```

Web runs on `http://web.localhost:8080`, worker in background.

## Multiple Apps

Edit `guvnor.yaml`:

```yaml
apps:
  - name: web
    command: npm
    args: ["start"]
  - name: api
    command: python
    args: ["api.py"]
```

Each app gets its own hostname.

## Production

```bash
guvnor start --domain myapp.com --email admin@myapp.com
```

Automatic HTTPS with Let's Encrypt.

## Daily Use

```bash
guvnor start      # Start apps
guvnor logs       # View logs  
guvnor logs -f    # Follow logs (real-time)
guvnor status     # Check process status
guvnor restart    # Restart all
guvnor stop       # Stop all

# 🆕 Certificate management
guvnor cert info    # Show certificate information
guvnor cert renew   # Renew expiring certificates
guvnor cert cleanup # Clean up expired certificates
```

## Config Priority

1. `guvnor.yaml` (primary)
2. `Procfile` (fallback)
3. Auto-detection

## Files

- `guvnor.yaml` - Main config
- `Procfile` - Foreman compatibility
- `.env` - Environment variables

## 🆕 Advanced Features

### Request Tracking
Enable distributed tracing across your microservices:

```yaml
# guvnor.yaml
server:
  enable_tracking: true
  tracking_header: "X-GUVNOR-TRACKING"
```

Track requests across services with UUID chains for complete observability.

### Certificate Headers
Inject client certificate information into your applications:

```yaml
# guvnor.yaml  
tls:
  certificate_headers: true
```

Your backend receives certificate details as HTTP headers for authentication and audit.

### Management API
Monitor and control your processes programmatically:

```bash
# Check process status
curl http://localhost:9080/api/status

# Stream logs
curl http://localhost:9080/api/logs?process=web-app&follow=true
```

## Docs

- [Config Reference](configuration.md)
- [Examples](examples.md)
- [Certificates & Tracking](certificates_and_tracking.md) 🆕
- [Production](systemd.md)
