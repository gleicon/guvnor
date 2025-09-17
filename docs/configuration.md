# Configuration Reference

Complete reference for `guvnor.yaml` configuration.

## Server Configuration

```yaml
server:
  http_port: 8080                    # Default: 8080
  https_port: 8443                   # Default: 8443  
  log_level: info                    # debug, info, warn, error
  
  # 🆕 Request Tracking Features
  enable_tracking: true              # Enable UUID request tracking
  tracking_header: "X-GUVNOR-TRACKING"  # Custom header name
  
  # Timeouts and Performance
  read_timeout: 30s                  # HTTP read timeout
  write_timeout: 30s                 # HTTP write timeout
  shutdown_timeout: 30s              # Graceful shutdown timeout
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

### 🆕 Certificate Header Injection (Valve-Inspired)

Guvnor can inject client certificate information as HTTP headers, similar to Apache's mod_ssl and valve systems:

```yaml
# Global certificate headers (affects all apps)
tls:
  certificate_headers: true   # Enable certificate header injection
  
apps:
  - name: secure-app
    tls:
      enabled: true
      certificate_headers: true  # Per-app override
```

**Injected Headers:**
- `X-Certificate-Detected`: "on" when client certificates present, "off" otherwise
- `X-Certificate-CN`: Formatted certificate subject (DN format)
- `X-Certificate-Subject`: Full certificate subject string
- `X-Certificate-Issuer`: Certificate issuer information
- `X-Certificate-Serial`: Certificate serial number
- `X-Certificate-Not-Before`: Certificate validity start date
- `X-Certificate-Not-After`: Certificate validity end date
- `X-Certificate-Status`: "valid" or "expired"

**Use Cases:**
- User identification based on client certificates
- Certificate-based authorization in backend applications
- Audit trails with certificate details
- Integration with existing authentication systems

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

## 🆕 Request Tracking Configuration

Guvnor provides advanced request tracking with UUID chains for distributed tracing:

```yaml
server:
  enable_tracking: true                    # Enable request tracking
  tracking_header: "X-GUVNOR-TRACKING"   # Custom header name (optional)
```

**How Request Tracking Works:**
1. **First request**: `X-GUVNOR-TRACKING: uuid1`
2. **Service calls another**: `X-GUVNOR-TRACKING: uuid1;uuid2`
3. **Third service call**: `X-GUVNOR-TRACKING: uuid1;uuid2;uuid3`

**Features:**
- UUID4 generation for unique request identification
- Chain-style tracking across microservices
- Included in Apache-style access logs
- Configurable header name for integration with existing systems

**Example Log Output:**
```
[::1] - - [14/Sep/2025:21:39:41 -0300] "GET /api/users" 200 1234 "-" "curl/8.15.0" app=api-service rt=45ms track=a1b2c3d4-e5f6-7890-abcd-ef1234567890;b2c3d4e5-f6g7-8901-bcde-f23456789012
```

## 🆕 Management API

Guvnor provides a REST API for monitoring and management:

```yaml
server:
  http_port: 8080        # Main proxy port
  # Management API runs on http_port + 1000 (e.g., 9080)
```

**Available Endpoints:**
- `GET /api/status` - Process status and health
- `GET /api/logs?process=name&lines=100` - Application logs
- `POST /api/stop` - Stop all processes
- `POST /api/restart` - Restart processes

**Example API Usage:**
```bash
# Get process status
curl http://localhost:9080/api/status

# Get logs for specific app
curl http://localhost:9080/api/logs?process=web-app&lines=50

# Stop all processes
curl -X POST http://localhost:9080/api/stop
```

## Configuration Generation

Use `guvnor init` to auto-generate configuration based on detected applications in the current directory. The generated configuration includes:

- Detected application types and commands
- Appropriate port assignments
- Framework-specific health check paths
- Development-friendly defaults with production-ready comments
- Request tracking and certificate headers configured