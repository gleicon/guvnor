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