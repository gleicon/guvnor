# API Documentation Example

Example of structured API-style documentation.

## Commands

### guvnor init

<div class="api-section">

<span class="api-method">INIT</span> <code class="api-endpoint">guvnor init [flags]</code>

Initialize Guvnor configuration in current directory.

</div>

<div class="command-section">
<div class="command-title">Description</div>
<div class="command-description">Auto-detects applications and generates guvnor.yaml configuration file.</div>

**Usage:**
```bash
guvnor init
guvnor init --force    # Overwrite existing config
```

**Detection Logic:**
1. Scans for `package.json`, `go.mod`, `Cargo.toml`, etc.
2. Identifies application type and optimal commands
3. Generates configuration with smart defaults

</div>

### guvnor start

<div class="api-section">

<span class="api-method">START</span> <code class="api-endpoint">guvnor start [app-name] [flags]</code>

Start applications defined in configuration.

</div>

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>Examples</strong>
</div>
<div class="config-content">

```bash
# Start all applications
guvnor start

# Start specific application
guvnor start webapp

# Start with production domain
guvnor start --domain myapp.com --email admin@myapp.com
```

</div>
</div>

<div>

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--domain` | string | Production domain for TLS |
| `--email` | string | Email for Let's Encrypt |
| `--config` | string | Custom config file path |
| `--port` | int | Override HTTP port |

</div>
</div>

## Configuration Schema

<div class="api-section">

<span class="api-method">YAML</span> <code class="api-endpoint">guvnor.yaml</code>

Complete configuration file schema.

</div>

<div class="config-section">
<div class="config-example">
<div class="config-header">
<strong>Server Configuration</strong>
</div>
<div class="config-content">

```yaml
server:
  http_port: 8080      # HTTP port
  https_port: 8443     # HTTPS port  
  log_level: info      # Log verbosity
```

</div>
</div>

<div class="config-example">
<div class="config-header">
<strong>Application Definition</strong>
</div>
<div class="config-content">

```yaml
apps:
  - name: webapp           # Required: unique identifier
    hostname: app.local    # Optional: routing hostname
    port: 3000            # Optional: auto-assigned
    command: node         # Required: executable
    args: ["server.js"]   # Optional: arguments
    working_dir: ./app    # Optional: working directory
    environment:          # Optional: env variables
      NODE_ENV: production
    health_check:         # Optional: monitoring
      enabled: true
      path: /health
      interval: 30s
    restart_policy:       # Optional: restart behavior
      enabled: true
      max_retries: 5
    tls:                  # Optional: HTTPS config
      enabled: true
      auto_cert: true
      email: admin@app.com
```

</div>
</div>
</div>

## Status Responses

### guvnor status

<div class="api-section">

<span class="api-method">STATUS</span> <code class="api-endpoint">Application Status</code>

Shows current state of all managed applications.

</div>

<div class="command-section">
<div class="command-title">Output Format</div>
<div class="command-description">Tabular display of application status</div>

```bash
$ guvnor status

APP         PID    STATUS    RESTARTS  PORT   UPTIME    COMMAND
web         1234   running   0         3000   2h 15m    node server.js
api         1235   running   1         8000   1h 30m    uvicorn main:app  
worker      1236   stopped   3         -      -         python worker.py
```

**Status Values:**
- `running` - Application is healthy and responding
- `starting` - Application is initializing  
- `stopped` - Application is not running
- `failed` - Application crashed and exceeded retry limit

</div>
</div>