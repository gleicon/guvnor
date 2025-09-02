# Architecture

How Guvnor works internally and the flow of configuration files.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Guvnor Process                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐    ┌──────────────────┐                   │
│  │  HTTP Server    │    │  Management API  │                   │
│  │  (8080/8443)    │    │  (REST/IPC)      │                   │
│  └─────────────────┘    └──────────────────┘                   │
│           │                       │                            │
│           ▼                       ▼                            │
│  ┌─────────────────┐    ┌──────────────────┐                   │
│  │ Reverse Proxy   │    │ Process Manager  │                   │
│  │ - Host routing  │    │ - Lifecycle mgmt │                   │
│  │ - TLS termination│   │ - Health checks  │                   │
│  │ - Load balancing│    │ - Restart policy │                   │
│  └─────────────────┘    └──────────────────┘                   │
│           │                       │                            │
│           │              ┌────────┼────────┐                   │
│           │              ▼        ▼        ▼                   │
│           │      ┌─────────┐ ┌─────────┐ ┌─────────┐           │
│           │      │ App A   │ │ App B   │ │ App C   │           │
│           │      │ :3000   │ │ :8000   │ │ :9000   │           │
│           │      └─────────┘ └─────────┘ └─────────┘           │
│           │              │        │        │                   │
│  ┌─────────────────┐     └────────┼────────┘                   │
│  │ TLS Manager     │              │                            │
│  │ - Let's Encrypt │     ┌────────▼────────┐                   │
│  │ - Auto renewal  │     │ Log Aggregator  │                   │
│  │ - Per-app certs │     │ - Circular buf  │                   │
│  └─────────────────┘     │ - Cross-process │                   │
│           │               └─────────────────┘                   │
└───────────┼─────────────────────────────────────────────────────┘
            │
    ┌───────▼────────┐
    │ Certificate    │
    │ Storage        │
    │ ./certs/       │
    └────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      External Traffic                          │
└─────────────────────────────────────────────────────────────────┘
            │
    ┌───────▼────────┐
    │ HTTP Request   │
    │ Host: api.app  │
    └───────┬────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Guvnor Proxy                                │
├─────────────────────────────────────────────────────────────────┤
│  1. Parse Host header                                           │
│  2. Match to app configuration                                  │
│  3. Apply TLS if configured                                     │
│  4. Forward to app port                                         │
│  5. Return response                                             │
└─────────────────────────────────────────────────────────────────┘
```

## Configuration Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Configuration Lifecycle                     │
└─────────────────────────────────────────────────────────────────┘

1. INITIALIZATION PHASE
   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
   │   Project   │    │ guvnor init │    │ Auto-detect │
   │ Directory   ├───►│   Command   ├───►│    Apps     │
   │             │    │             │    │             │
   └─────────────┘    └─────────────┘    └─────┬───────┘
                                               │
   ┌─────────────┐    ┌─────────────┐         │
   │   Procfile  │    │    .env     │         │
   │ (optional)  │    │ (optional)  │         │
   │             │    │             │         │
   └─────────────┘    └─────────────┘         │
                                               │
                      ┌─────────────┐         │
                      │guvnor.yaml  │◄────────┘
                      │ Generated   │
                      │             │
                      └─────────────┘

2. RUNTIME PHASE
   ┌─────────────┐
   │guvnor start │
   │   Command   │
   └─────┬───────┘
         │
         ▼
   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
   │Load Config  │    │ Validate    │    │   Start     │
   │guvnor.yaml  ├───►│   Apps      ├───►│ Processes   │
   │             │    │             │    │             │
   └─────────────┘    └─────────────┘    └─────┬───────┘
                                               │
   ┌─────────────┐    ┌─────────────┐         │
   │   .env      │    │Environment  │         │
   │Variables    ├───►│Substitution │         │
   │             │    │             │         │
   └─────────────┘    └─────────────┘         │
                                               │
   ┌─────────────┐    ┌─────────────┐         │
   │   Procfile  │    │Process Defs │         │
   │Parsing      ├───►│(if exists)  │         │
   │(fallback)   │    │             │         │
   └─────────────┘    └─────────────┘         │
                                               │
                      ┌─────────────┐         │
                      │   Runtime   │◄────────┘
                      │ Supervision │
                      └─────────────┘

3. MANAGEMENT PHASE
   ┌─────────────┐
   │ CLI Commands│
   │ guvnor logs │
   │ guvnor stop │
   └─────┬───────┘
         │
         ▼
   ┌─────────────┐    ┌─────────────┐
   │Management   │    │   Live      │
   │    API      ├───►│ Process     │
   │             │    │ Control     │
   └─────────────┘    └─────────────┘
```

## File Priorities and Lifecycle

```
Configuration Priority (highest to lowest):
┌─────────────────────────────────────────┐
│ 1. guvnor.yaml (explicit config)       │ ← Primary
├─────────────────────────────────────────┤
│ 2. Procfile (process definitions)      │ ← Fallback  
├─────────────────────────────────────────┤
│ 3. Auto-detection (package.json, etc.) │ ← Last resort
└─────────────────────────────────────────┘

Environment Variables:
┌─────────────────────────────────────────┐
│ 1. guvnor.yaml app.environment{}        │ ← Per-app override
├─────────────────────────────────────────┤  
│ 2. .env file variables                  │ ← Global defaults
├─────────────────────────────────────────┤
│ 3. System environment                   │ ← System-wide
└─────────────────────────────────────────┘

File Lifecycle:
┌────────────┬─────────────┬──────────────┬─────────────┐
│    File    │   Created   │    Read      │   Updated   │
├────────────┼─────────────┼──────────────┼─────────────┤
│guvnor.yaml │ guvnor init │ guvnor start │ Manual edit │
│            │             │ guvnor validate │          │
├────────────┼─────────────┼──────────────┼─────────────┤
│ Procfile   │ guvnor init │ guvnor start │ Manual edit │
│            │ (optional)  │ (if no yaml) │             │
├────────────┼─────────────┼──────────────┼─────────────┤
│   .env     │ guvnor init │ guvnor start │ Manual edit │
│            │ (optional)  │ (always)     │             │
├────────────┼─────────────┼──────────────┼─────────────┤
│   logs     │ guvnor start│ guvnor logs  │ Auto-rotate │
└────────────┴─────────────┴──────────────┴─────────────┘
```

## Process Management

```
┌─────────────────────────────────────────────────────────────────┐
│                     Process Supervision                        │
└─────────────────────────────────────────────────────────────────┘

Application Lifecycle:
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   STOPPED   │───►│  STARTING   │───►│   RUNNING   │
│             │    │             │    │             │
└─────────────┘    └─────┬───────┘    └─────┬───────┘
        ▲                │                  │
        │                ▼                  │
        │          ┌─────────────┐          │
        │          │   FAILED    │          │
        │          │             │          │
        │          └─────┬───────┘          │
        │                │                  │
        │                ▼                  │
        │          ┌─────────────┐          │
        └──────────┤ RESTARTING  │◄─────────┘
                   │             │
                   └─────────────┘

Health Check Flow:
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│HTTP Request │    │   Success   │    │   Healthy   │
│to /health   ├───►│ (200 OK)    ├───►│    State    │
│             │    │             │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
                           │
                           ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Failure   │    │   Retry     │    │  Unhealthy  │
│ (Non-200)   ├───►│ (N times)   ├───►│  -> Restart │
│             │    │             │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
```

## Virtual Host Routing

```
┌─────────────────────────────────────────────────────────────────┐
│                    Request Routing                             │
└─────────────────────────────────────────────────────────────────┘

Incoming Request:
┌─────────────────┐
│ GET /api/users  │
│ Host: api.app   │
│ User-Agent: ... │
└─────┬───────────┘
      │
      ▼
┌─────────────────┐
│ Parse Host      │
│ header          │
│ "api.app"       │
└─────┬───────────┘
      │
      ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Match against   │    │   app: api      │    │ Forward to      │
│ app hostnames   ├───►│ hostname:api.app├───►│ localhost:8000  │
│                 │    │ port: 8000      │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                      │
      ┌───────────────────────────────────────────────┘
      ▼
┌─────────────────┐    ┌─────────────────┐
│ App Response    │    │ Return to       │
│ 200 OK          ├───►│ Client          │
│ {...users...}   │    │                 │
└─────────────────┘    └─────────────────┘

Multiple Apps:
web.localhost:8080 ──► App A (port 3000) ──► Frontend
api.localhost:8080 ──► App B (port 8000) ──► API
admin.localhost:8080 ► App C (port 9000) ──► Admin
```

## TLS Certificate Management

```
┌─────────────────────────────────────────────────────────────────┐
│                   TLS Certificate Flow                         │
└─────────────────────────────────────────────────────────────────┘

Per-App TLS Configuration:
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   App A     │    │   App B     │    │   App C     │
│ hostname:   │    │ hostname:   │    │ hostname:   │
│ web.com     │    │ api.com     │    │ admin.com   │
│ tls: true   │    │ tls: true   │    │ tls: false  │
└─────┬───────┘    └─────┬───────┘    └─────┬───────┘
      │                  │                  │
      ▼                  ▼                  ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│Let's Encrypt│    │Let's Encrypt│    │   HTTP      │
│Certificate  │    │Certificate  │    │   Only      │
│web.com      │    │api.com      │    │             │
└─────────────┘    └─────────────┘    └─────────────┘

Certificate Lifecycle:
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Request    │    │   Obtain    │    │   Store     │
│Certificate  ├───►│from LE/ACME ├───►│ in cert_dir │
│             │    │             │    │             │
└─────────────┘    └─────────────┘    └─────┬───────┘
        ▲                                   │
        │                                   ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│Auto-renewal │◄───┤   Monitor   │◄───┤    Load     │
│(30 days)    │    │  Expiry     │    │   & Use     │
│             │    │             │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
```

## Key Components

### 1. Discovery Engine
- Scans project directory for known application patterns
- Detects package.json, go.mod, Cargo.toml, requirements.txt
- Generates appropriate commands and configurations

### 2. Process Manager  
- Supervises application processes using Go's os/exec
- Implements restart policies with exponential backoff
- Monitors process health via HTTP/TCP checks
- Handles graceful shutdown (SIGTERM → SIGKILL)

### 3. Reverse Proxy
- HTTP server with virtual host routing
- Parses Host headers to route requests
- Handles TLS termination per-application
- Implements load balancing for multiple instances

### 4. Certificate Manager
- Integrates with Let's Encrypt ACME protocol
- Manages per-application certificates
- Automatic renewal before expiry
- Stores certificates in configurable directory

### 5. Configuration System
- YAML-based primary configuration (guvnor.yaml)
- Procfile fallback for Heroku compatibility
- Environment variable substitution from .env files
- Smart defaults with override capability

### 6. Logging System
- Circular buffer for cross-process log aggregation
- HTTP API for log retrieval by CLI commands
- Structured logging with configurable levels
- Process-specific log filtering and display

This architecture provides a single-binary solution that replaces complex infrastructure while maintaining flexibility and production-readiness.