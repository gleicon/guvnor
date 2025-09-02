# Configuration Reference

Complete reference for `guvnor.yaml` configuration.

## Server Configuration

```yaml
server:
  http_port: 8080      # Default: 8080
  https_port: 8443     # Default: 8443  
  log_level: info      # debug, info, warn, error
```

## Application Configuration

### Required Parameters

```yaml
apps:
  - name: webapp       # Required: unique identifier
    command: node      # Required: command to run
```

### Optional Parameters

```yaml
apps:
  - name: webapp
    hostname: web.localhost    # Default: {name}.localhost
    port: 3000                # Default: auto-assigned
    args: ["server.js"]       # Default: []
    working_dir: ./app        # Default: current dir
```

## Multi-App Configuration

```yaml
server:
  http_port: 8080
  https_port: 8443

apps:
  # Frontend application
  - name: frontend
    hostname: web.localhost     # Routes requests to this app
    port: 3000
    command: npm
    args: ["start"]
    
  # API backend
  - name: api
    hostname: api.localhost     # Different hostname
    port: 8000
    command: uvicorn
    args: ["main:app", "--host", "0.0.0.0", "--port", "8000"]
    tls:
      enabled: true
      auto_cert: true
      
  # Admin panel with auto-generated hostname (admin.localhost)
  - name: admin
    port: 9000
    command: python
    args: ["-m", "streamlit", "run", "admin.py"]
```

## Production Configuration

```yaml
server:
  http_port: 80               # Standard ports for production
  https_port: 443
  log_level: warn             # Reduce log verbosity

apps:
  - name: web-app
    hostname: myapp.com       # Production domain
    port: 3000
    command: node
    args: ["dist/server.js"]  # Production build
    
    tls:
      enabled: true
      auto_cert: true         # Let's Encrypt certificates
      email: admin@myapp.com
      staging: false          # Production certificates
      
    health_check:
      enabled: true
      path: /health
      interval: 30s
      timeout: 10s
      retries: 3
      
    restart_policy:
      enabled: true
      max_retries: 10
      backoff: 10s

# Global settings
tls:
  enabled: true
  cert_dir: /var/lib/guvnor/certs
  force_https: true           # Redirect HTTP to HTTPS
```

## Smart Defaults

When values are omitted, guvnor uses intelligent defaults:

- **hostname**: Auto-generated as `{app-name}.localhost`
- **port**: Auto-assigned starting from 3000
- **working_dir**: Current directory
- **tls.email**: Inherits from global TLS settings
- **health_check.path**: `/` for most apps, framework-specific for detected apps
- **restart_policy**: Enabled with reasonable defaults

## Environment Variables

```yaml
apps:
  - name: my-app
    environment:
      NODE_ENV: production
      DATABASE_URL: postgres://localhost/mydb
      PORT: "3000"            # Always use strings for port numbers
```

## Health Checks

```yaml
apps:
  - name: web-app
    health_check:
      enabled: true
      path: /health           # HTTP endpoint to check
      interval: 30s           # How often to check
      timeout: 5s             # Request timeout
      retries: 3              # Failures before marking unhealthy
      expected_status: 200    # Expected HTTP status code
```

## TLS Configuration

### Per-App TLS
```yaml
apps:
  - name: secure-app
    hostname: secure.example.com
    tls:
      enabled: true
      auto_cert: true         # Automatic Let's Encrypt
      email: admin@example.com
      staging: false          # Use production certificates
```

### Manual Certificates
```yaml
apps:
  - name: custom-cert-app
    tls:
      enabled: true
      auto_cert: false        # Manual certificate management
      cert_file: /path/to/cert.pem
      key_file: /path/to/key.pem
```

## Restart Policies

```yaml
apps:
  - name: critical-app
    restart_policy:
      enabled: true
      max_retries: 10         # Maximum restart attempts
      backoff: 5s             # Delay between restarts
      backoff_multiplier: 2.0 # Exponential backoff
      max_backoff: 300s       # Maximum backoff delay
```

## Configuration Validation

Guvnor validates configuration on startup. Common validation rules:

- App names must be unique
- Ports must not conflict between apps
- Hostnames should be valid domain names
- TLS email required when auto_cert is enabled
- Working directories must exist

Run `guvnor validate` to check configuration without starting apps.

## Configuration Generation

Use `guvnor init` to auto-generate configuration based on detected applications in the current directory. The generated configuration includes:

- Detected application types and commands
- Appropriate port assignments
- Framework-specific health check paths
- Development-friendly defaults with production-ready comments