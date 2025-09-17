# Examples

Real-world configuration examples for common scenarios.

## Node.js App

```yaml
apps:
  - name: webapp
    hostname: app.localhost
    command: npm
    args: ["start"]
    environment:
      NODE_ENV: development
      PORT: "3000"
```

## Python FastAPI

```yaml
apps:
  - name: api
    hostname: api.localhost
    command: uvicorn
    args: ["main:app", "--host", "0.0.0.0", "--reload"]
    environment:
      DATABASE_URL: sqlite:///./test.db
```

## Go Application

```yaml
apps:
  - name: server
    hostname: go.localhost
    command: go
    args: ["run", "main.go"]
    working_dir: ./cmd/server
```

## Multi-Service Architecture

```yaml
# guvnor.yaml
server:
  http_port: 8080
  https_port: 8443

apps:
  # Frontend React app
  - name: frontend
    hostname: web.localhost
    port: 3000
    command: npm
    args: ["start"]
    
  # API backend
  - name: api
    hostname: api.localhost
    port: 8000
    command: uvicorn
    args: ["main:app", "--host", "0.0.0.0", "--port", "8000"]
    
  # Background worker
  - name: worker
    port: 8001
    command: celery
    args: ["-A", "app", "worker"]
    
  # Redis server
  - name: redis
    port: 6379
    command: redis-server
    args: ["--port", "6379"]
```

## Real-World Production Example

```yaml
# Production multi-app setup
server:
  http_port: 80
  https_port: 443
  log_level: warn

apps:
  # Main web application
  - name: webapp
    hostname: mycompany.com
    port: 3000
    command: node
    args: ["dist/server.js"]
    environment:
      NODE_ENV: production
    tls:
      enabled: true
      auto_cert: true
      email: admin@mycompany.com
      staging: false
    health_check:
      enabled: true
      path: /health
      interval: 30s
    restart_policy:
      enabled: true
      max_retries: 10
      backoff: 10s

  # API service
  - name: api
    hostname: api.mycompany.com
    port: 8000
    command: gunicorn
    args: ["app.wsgi:application", "--bind", "0.0.0.0:8000", "--workers", "4"]
    environment:
      DJANGO_SETTINGS_MODULE: app.settings.production
    tls:
      enabled: true
      auto_cert: true
      email: api@mycompany.com
    health_check:
      enabled: true
      path: /api/health

  # Admin interface
  - name: admin
    hostname: admin.mycompany.com
    port: 9000
    command: python
    args: ["-m", "streamlit", "run", "admin_dashboard.py"]
    tls:
      enabled: true
      auto_cert: true
      email: admin@mycompany.com

# Global TLS settings
tls:
  enabled: true
  cert_dir: /var/lib/guvnor/certs
  force_https: true
```

## Microservices Example

```yaml
# Microservices with service discovery
apps:
  # API Gateway
  - name: gateway
    hostname: gateway.localhost
    port: 8080
    command: node
    args: ["gateway.js"]
    
  # Authentication service
  - name: auth-service
    hostname: auth.localhost
    port: 8001
    command: python
    args: ["-m", "auth.main"]
    
  # User service
  - name: user-service
    hostname: users.localhost  
    port: 8002
    command: python
    args: ["-m", "users.main"]
    
  # Order service
  - name: order-service
    hostname: orders.localhost
    port: 8003
    command: go
    args: ["run", "cmd/server/main.go"]
    
  # Notification service
  - name: notifications
    hostname: notifications.localhost
    port: 8004
    command: ./target/release/notifications
```

## Development with Hot Reload

```yaml
# Development configuration with auto-reload
apps:
  # Next.js with hot reload
  - name: frontend
    hostname: web.localhost
    command: npm
    args: ["run", "dev"]
    
  # FastAPI with auto-reload
  - name: api
    hostname: api.localhost
    command: uvicorn
    args: ["main:app", "--reload", "--host", "0.0.0.0", "--port", "8000"]
    
  # Rust with cargo watch
  - name: service
    hostname: service.localhost
    command: cargo
    args: ["watch", "-x", "run"]
```

## Docker Integration

```yaml
# Mix of native processes and containers
apps:
  # Native Node.js app
  - name: webapp
    command: node
    args: ["server.js"]
    port: 3000
    
  # Containerized database
  - name: postgres
    command: docker
    args: ["run", "--rm", "-p", "5432:5432", "-e", "POSTGRES_PASSWORD=secret", "postgres:15"]
    port: 5432
    
  # Redis container
  - name: redis
    command: docker
    args: ["run", "--rm", "-p", "6379:6379", "redis:7"]
    port: 6379
```

## Complex SaaS Application

```yaml
# Full SaaS deployment
server:
  http_port: 80
  https_port: 443

apps:
  # Frontend application
  - name: web
    hostname: app.saascompany.com
    port: 3000
    command: node
    args: ["dist/server.js"]
    environment:
      NODE_ENV: production
      API_URL: https://api.saascompany.com
    tls:
      enabled: true
      auto_cert: true
      email: ops@saascompany.com
    
  # Main API
  - name: api
    hostname: api.saascompany.com
    port: 8000
    command: gunicorn
    args: ["app.wsgi:application", "--bind", "0.0.0.0:8000", "--workers", "8"]
    environment:
      DATABASE_URL: postgres://user:pass@db.internal:5432/app
      REDIS_URL: redis://cache.internal:6379
    tls:
      enabled: true
      auto_cert: true
      email: api@saascompany.com
    health_check:
      enabled: true
      path: /health
      interval: 15s
    
  # Background job processor
  - name: worker-high
    port: 8001
    command: celery
    args: ["-A", "app", "worker", "-Q", "high_priority", "--concurrency=4"]
    restart_policy:
      enabled: true
      max_retries: 5
      
  - name: worker-low
    port: 8002  
    command: celery
    args: ["-A", "app", "worker", "-Q", "low_priority", "--concurrency=2"]
    
  # Scheduled tasks
  - name: scheduler
    port: 8003
    command: celery
    args: ["-A", "app", "beat"]
    
  # Admin dashboard
  - name: admin
    hostname: admin.saascompany.com
    port: 9000
    command: python
    args: ["manage.py", "runserver", "0.0.0.0:9000"]
    environment:
      DJANGO_SETTINGS_MODULE: app.settings.admin
    tls:
      enabled: true
      auto_cert: true
      email: admin@saascompany.com

tls:
  enabled: true
  cert_dir: /var/lib/guvnor/certs
  force_https: true
```

## CLI Usage Examples

```bash
# Basic operations
guvnor init                    # Generate configuration
guvnor start                   # Start all apps
guvnor start webapp api        # Start specific apps  
guvnor stop worker             # Stop specific app
guvnor restart api             # Restart app
guvnor status                  # Show all app status
guvnor status webapp           # Show specific app status
guvnor logs                    # View all logs
guvnor logs webapp             # View specific app logs
guvnor validate                # Validate configuration

# Production deployment
guvnor start --domain myapp.com --email admin@myapp.com

# Development with debug logging
guvnor start --log-level debug
```

## Health Check Examples

```yaml
apps:
  # HTTP health check
  - name: api
    health_check:
      enabled: true
      path: /api/v1/health
      interval: 30s
      timeout: 5s
      retries: 3
      expected_status: 200
      
  # Custom command health check
  - name: worker
    health_check:
      enabled: true
      command: ["python", "health_check.py"]
      interval: 60s
      timeout: 10s
      
  # TCP port check
  - name: database
    health_check:
      enabled: true
      type: tcp
      port: 5432
      interval: 10s
```

## Environment-Specific Configurations

### Development
```yaml
# dev.guvnor.yaml
server:
  log_level: debug
  
apps:
  - name: webapp
    command: npm
    args: ["run", "dev"]
    environment:
      NODE_ENV: development
      API_URL: http://api.localhost:8080
```

### Production  
```yaml
# prod.guvnor.yaml
server:
  http_port: 80
  https_port: 443
  log_level: warn
  
apps:
  - name: webapp
    hostname: myapp.com
    command: node
    args: ["dist/server.js"]
    environment:
      NODE_ENV: production
    tls:
      enabled: true
      auto_cert: true
      email: ops@myapp.com
```

Load specific configuration:
```bash
guvnor start --config dev.guvnor.yaml
guvnor start --config prod.guvnor.yaml
```

## 🆕 Advanced Features Examples

### Request Tracking & Certificate Headers

```yaml
# Enterprise configuration with full observability
server:
  http_port: 8080
  https_port: 8443
  log_level: info
  
  # 🆕 Request tracking for distributed tracing
  enable_tracking: true
  tracking_header: "X-REQUEST-ID"   # Custom header name

apps:
  # Frontend with certificate-based auth
  - name: secure-portal
    hostname: portal.company.com
    port: 3000
    command: node
    args: ["server.js"]
    tls:
      enabled: true
      auto_cert: true
      email: security@company.com
      certificate_headers: true     # 🆕 Inject certificate info
    environment:
      # Your app receives these headers:
      # X-Certificate-Detected: on/off
      # X-Certificate-CN: CN=John Doe,OU=IT,O=Company
      # X-Certificate-Subject: full subject string
      # X-Certificate-Serial: certificate serial number
      NODE_ENV: production

  # API service with request tracking  
  - name: api-gateway
    hostname: api.company.com
    port: 8000
    command: uvicorn
    args: ["gateway.main:app", "--host", "0.0.0.0", "--port", "8000"]
    tls:
      enabled: true
      auto_cert: true
      certificate_headers: true
    environment:
      # Your app receives tracking headers:
      # X-REQUEST-ID: uuid1;uuid2;uuid3 (chain of UUIDs)
      LOG_LEVEL: info

# Global certificate header injection
tls:
  certificate_headers: true         # 🆕 Enable globally
  enabled: true
  cert_dir: /var/lib/guvnor/certs
  force_https: true
```

### Microservices with Full Observability

```yaml
# Complete microservices setup with tracking
server:
  enable_tracking: true
  tracking_header: "X-TRACE-ID"

apps:
  # API Gateway - entry point for all requests
  - name: gateway
    hostname: gateway.company.com
    port: 8080
    command: node
    args: ["gateway.js"]
    tls:
      enabled: true
      auto_cert: true
    # Receives: X-TRACE-ID: uuid1
    # Forwards: X-TRACE-ID: uuid1;uuid2
    
  # User Service - handles authentication
  - name: user-service
    hostname: users.company.com
    port: 8001
    command: python
    args: ["-m", "users.main"]
    tls:
      enabled: true
      auto_cert: true
      certificate_headers: true     # Gets client cert info
    # Receives: X-TRACE-ID: uuid1;uuid2
    # Forwards: X-TRACE-ID: uuid1;uuid2;uuid3
    
  # Payment Service - secure transactions
  - name: payment-service
    hostname: payments.company.com
    port: 8002
    command: go
    args: ["run", "cmd/payments/main.go"]
    tls:
      enabled: true
      auto_cert: true
      certificate_headers: true     # Critical for payment security
    # Receives: X-TRACE-ID: uuid1;uuid2;uuid3
    # Can validate client certificates via headers

tls:
  certificate_headers: true
```

### Real-Time Monitoring Setup

```yaml
# Production setup with comprehensive monitoring
server:
  http_port: 80
  https_port: 443
  enable_tracking: true             # 🆕 Track all requests
  log_level: info

apps:
  # Main application
  - name: webapp
    hostname: myapp.com
    port: 3000
    command: node
    args: ["dist/server.js"]
    tls:
      enabled: true
      auto_cert: true
      email: ops@myapp.com
    health_check:
      enabled: true
      path: /health
      interval: 15s
    restart_policy:
      enabled: true
      max_retries: 5

  # Monitoring dashboard
  - name: monitoring
    hostname: monitor.myapp.com
    port: 3001
    command: node
    args: ["monitor/server.js"]
    tls:
      enabled: true
      auto_cert: true
    environment:
      # Monitor receives tracking data from logs
      GUVNOR_API_URL: http://localhost:9080/api
```

### Certificate Management Examples

```bash
# Certificate management commands
guvnor cert info                    # Show all certificates
guvnor cert info secure.myapp.com   # Show specific domain

# Certificate output example:
# DOMAIN                     STATUS      NOT BEFORE           NOT AFTER            PATH
# secure.myapp.com          valid       2025-01-15 10:30     2025-04-15 10:30     /var/lib/guvnor/certs/secure.myapp.com.crt
# api.myapp.com             expiring    2025-01-01 08:00     2025-09-20 08:00     /var/lib/guvnor/certs/api.myapp.com.crt

guvnor cert renew                   # Renew expiring certificates
guvnor cert cleanup                 # Remove expired certificates
```

### Request Tracking in Application Code

**Node.js Example:**
```javascript
// Your Node.js app receives tracking headers
app.use((req, res, next) => {
  const traceId = req.headers['x-request-id'];
  const certDetected = req.headers['x-certificate-detected'];
  const certCN = req.headers['x-certificate-cn'];
  
  console.log(`Request ${traceId} from ${certCN || 'anonymous'}`);
  
  // When making downstream requests, forward the header
  axios.get('http://api-service', {
    headers: { 'X-Request-ID': traceId }
  });
  
  next();
});
```

**Python FastAPI Example:**
```python
from fastapi import FastAPI, Request
import httpx

app = FastAPI()

@app.middleware("http")
async def tracking_middleware(request: Request, call_next):
    trace_id = request.headers.get("x-request-id")
    cert_cn = request.headers.get("x-certificate-cn")
    
    print(f"Request {trace_id} from {cert_cn or 'anonymous'}")
    
    # Forward tracking header to downstream services
    async with httpx.AsyncClient() as client:
        response = await client.get(
            "http://another-service",
            headers={"X-Request-ID": trace_id}
        )
    
    return await call_next(request)
```

### Advanced Log Analysis

With request tracking enabled, your logs include complete request journeys:

```bash
# Example log entries showing request flow:
[2025-09-15T00:45:12] "GET /api/users" 200 app=gateway rt=5ms track=a1b2c3d4-e5f6-7890-abcd-ef1234567890
[2025-09-15T00:45:12] "GET /users/profile" 200 app=user-service rt=15ms track=a1b2c3d4-e5f6-7890-abcd-ef1234567890;b2c3d4e5-f6g7-8901-bcde-f23456789012
[2025-09-15T00:45:12] "GET /payments/history" 200 app=payment-service rt=25ms track=a1b2c3d4-e5f6-7890-abcd-ef1234567890;b2c3d4e5-f6g7-8901-bcde-f23456789012;c3d4e5f6-g7h8-9012-cdef-345678901234

# Use grep/awk to trace complete request journeys:
grep "a1b2c3d4-e5f6-7890-abcd-ef1234567890" /var/log/guvnor.log
```

### Production Security Configuration

```yaml
# High-security production setup
server:
  http_port: 80
  https_port: 443
  enable_tracking: true
  tracking_header: "X-REQUEST-ID"
  log_level: warn

apps:
  - name: secure-api
    hostname: secure-api.company.com
    port: 8000
    command: gunicorn
    args: ["app.wsgi:application", "--bind", "0.0.0.0:8000"]
    tls:
      enabled: true
      auto_cert: true
      email: security@company.com
      certificate_headers: true     # Required for client cert auth
      staging: false                # Production certificates only
    environment:
      # Your app validates clients via certificate headers
      REQUIRE_CLIENT_CERT: "true"
      TRUSTED_CERT_ISSUERS: "CN=Company CA,O=Company Inc"
    health_check:
      enabled: true
      path: /security/health
      interval: 10s
    restart_policy:
      enabled: true
      max_retries: 3

tls:
  certificate_headers: true
  force_https: true                 # Redirect all HTTP to HTTPS
  cert_dir: /var/lib/guvnor/certs
```