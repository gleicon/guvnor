# Configuration Reference

Complete reference for `guvnor.yaml` configuration file.

## Server Configuration

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>server</strong> - Global server settings
</div>
<div class="config-content">

```yaml
server:
  http_port: 8080      # HTTP port (default: 8080)
  https_port: 8443     # HTTPS port (default: 8443)  
  log_level: info      # Log level: debug, info, warn, error
```

</div>
</div>

<div>

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `http_port` | integer | `8080` | Port for HTTP traffic |
| `https_port` | integer | `8443` | Port for HTTPS traffic |
| `log_level` | string | `info` | Logging verbosity |

### Examples

<div class="command-section">
<div class="command-title">Development</div>
<div class="command-description">Use non-privileged ports for local development</div>

```yaml
server:
  http_port: 8080
  https_port: 8443
  log_level: debug
```

</div>

<div class="command-section">
<div class="command-title">Production</div>
<div class="command-description">Use standard ports with reduced logging</div>

```yaml
server:
  http_port: 80
  https_port: 443
  log_level: warn
```

</div>

</div>
</div>

## Application Configuration

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>apps</strong> - Application definitions
</div>
<div class="config-content">

```yaml
apps:
  - name: web-app
    hostname: web.localhost
    port: 3000
    command: node
    args: ["server.js"]
    working_dir: /path/to/app
```

</div>
</div>

<div>

### Required Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Unique app identifier |
| `command` | string | Command to execute |

### Optional Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `hostname` | string | `{name}.localhost` | Hostname for routing |
| `port` | integer | auto-assigned | Port number |
| `args` | array | `[]` | Command arguments |
| `working_dir` | string | current dir | Working directory |

</div>
</div>

## Environment Variables

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>environment</strong> - Per-app environment
</div>
<div class="config-content">

```yaml
apps:
  - name: api
    environment:
      NODE_ENV: production
      DATABASE_URL: postgres://localhost/db
      PORT: "8000"
```

</div>
</div>

<div>

### Priority Order

1. **App-level environment** (highest priority)
2. **`.env` file variables**
3. **System environment** (lowest priority)

### Variable Substitution

Environment variables are substituted in command arguments:

```yaml
apps:
  - name: api
    args: ["--port", "$PORT"]
    environment:
      PORT: "8000"
```

</div>
</div>

## Health Checks

<div class="api-section">

<span class="api-method">GET</span> <code class="api-endpoint">/health</code>

Configure health monitoring for automatic restarts.

</div>

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>health_check</strong> - Health monitoring
</div>
<div class="config-content">

```yaml
apps:
  - name: api
    health_check:
      enabled: true
      path: /health
      interval: 30s
      timeout: 5s
      retries: 3
      expected_status: 200
```

</div>
</div>

<div>

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable health checks |
| `path` | string | `/` | HTTP endpoint path |
| `interval` | duration | `30s` | Check frequency |
| `timeout` | duration | `5s` | Request timeout |
| `retries` | integer | `3` | Failures before restart |
| `expected_status` | integer | `200` | Expected HTTP status |

</div>
</div>

## TLS Configuration

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>tls</strong> - HTTPS and certificates
</div>
<div class="config-content">

```yaml
# Per-app TLS
apps:
  - name: web
    hostname: myapp.com
    tls:
      enabled: true
      auto_cert: true
      email: admin@myapp.com
      staging: false

# Global TLS
tls:
  enabled: true
  cert_dir: ./certs
  force_https: true
```

</div>
</div>

<div>

### Per-App TLS

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable HTTPS |
| `auto_cert` | boolean | `false` | Use Let's Encrypt |
| `email` | string | required | Email for certificates |
| `staging` | boolean | `false` | Use staging certificates |

### Global TLS

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | boolean | `false` | Global HTTPS default |
| `cert_dir` | string | `./certs` | Certificate storage |
| `force_https` | boolean | `false` | Redirect HTTP→HTTPS |

</div>
</div>

## Complete Example

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>Multi-app production setup</strong>
</div>
<div class="config-content">

```yaml
server:
  http_port: 80
  https_port: 443
  log_level: warn

apps:
  - name: web
    hostname: myapp.com
    port: 3000
    command: node
    args: ["dist/server.js"]
    environment:
      NODE_ENV: production
    tls:
      enabled: true
      auto_cert: true
      email: admin@myapp.com
    health_check:
      enabled: true
      path: /health
      
  - name: api
    hostname: api.myapp.com
    port: 8000
    command: uvicorn
    args: ["main:app", "--host", "0.0.0.0"]
    tls:
      enabled: true
      auto_cert: true
      email: api@myapp.com
    restart_policy:
      enabled: true
      max_retries: 5

tls:
  enabled: true
  cert_dir: /var/lib/guvnor/certs
  force_https: true
```

</div>
</div>

<div>

This configuration runs two applications:

- **Web app** at `https://myapp.com` with health monitoring
- **API service** at `https://api.myapp.com` with auto-restart

Both use automatic Let's Encrypt certificates.

</div>
</div>

## Validation

<div class="command-section">
<div class="command-title">Check Configuration</div>
<div class="command-description">Validate your guvnor.yaml before starting</div>

```bash
guvnor validate
```

Common validation errors:
- Duplicate app names
- Port conflicts  
- Invalid hostnames
- Missing required fields

</div>