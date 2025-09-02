# Common Workflows

Daily workflows and common tasks with Guvnor.

## Development Workflow

### Starting a New Day
```bash
cd my-project
guvnor start      # Start all configured apps
guvnor status     # Check everything is running

# Visit your apps:
# http://web.localhost:8080
# http://api.localhost:8080
```

### Making Changes
```bash
# Code changes are live (if your app supports hot reload)
# For apps that need restart:
guvnor restart api-service

# View logs for debugging
guvnor logs api-service
guvnor logs         # All apps
```

### End of Day
```bash
guvnor stop        # Stop all apps
# Or just close terminal (apps stop automatically)
```

## Adding New Services

### Add Database to Existing Project
```bash
# Edit guvnor.yaml
apps:
  - name: web
    # ... existing config
    
  - name: database
    port: 5432
    command: postgres
    args: ["-D", "/usr/local/var/postgres"]
    # No hostname - internal service only

guvnor validate    # Check config
guvnor start       # Starts all apps including database
```

### Add Background Worker
```bash
# guvnor.yaml
apps:
  - name: worker
    # No hostname - background only
    port: 8001
    command: python
    args: ["worker.py"]
    environment:
      WORKER_QUEUE: "high_priority"
```

## Multiple Projects Workflow

### Switching Between Projects
```bash
# Project A
cd project-a
guvnor start       # Runs on localhost:8080

# Switch to Project B (different terminal/directory)
cd ../project-b
guvnor start       # ERROR: Port 8080 in use

# Solution 1: Stop first project
cd ../project-a && guvnor stop
cd ../project-b && guvnor start

# Solution 2: Use different ports
cd project-b
# Edit guvnor.yaml: server.http_port: 8090
guvnor start       # Runs on localhost:8090
```

### Running Multiple Projects Simultaneously
```bash
# Configure different ports for each project
# Project A: guvnor.yaml → http_port: 8080
# Project B: guvnor.yaml → http_port: 8090
# Project C: guvnor.yaml → http_port: 8100

# Access:
# http://app-a.localhost:8080
# http://app-b.localhost:8090  
# http://app-c.localhost:8100
```

## Production Workflows

### Initial Deployment
```bash
# On server
git clone your-repo
cd your-repo
guvnor init        # Generate config if needed

# Production start
guvnor start --domain myapp.com --email admin@myapp.com

# Check status
guvnor status
guvnor logs
```

### Updates and Deployments
```bash
# Update code
git pull

# Validate new config
guvnor validate

# Zero-downtime restart
guvnor restart

# Or restart specific service
guvnor restart api-service
```

### Rollback
```bash
git checkout previous-version
guvnor validate
guvnor restart
```

## Debugging Workflows

### App Won't Start
```bash
guvnor validate    # Check config syntax
guvnor logs        # See error messages
guvnor status      # Check which apps failed

# Start single app for debugging
guvnor start web-app
```

### Port Conflicts
```bash
# Find what's using the port
lsof -i :8080

# Or change Guvnor's port
# Edit guvnor.yaml: server.http_port: 8090
guvnor restart
```

### SSL/TLS Issues
```bash
guvnor logs        # Check certificate errors

# Test in staging first
guvnor start --domain staging.myapp.com --staging

# Check certificate files
ls -la certs/
```

### Performance Issues
```bash
# Check process status
guvnor status

# Monitor logs in real-time
guvnor logs -f

# Restart unhealthy services
guvnor restart slow-service
```

## Configuration Workflows

### Adding Environment Variables
```bash
# Method 1: Edit .env file
echo "API_KEY=secret123" >> .env
guvnor restart

# Method 2: Edit guvnor.yaml
apps:
  - name: api
    environment:
      API_KEY: "secret123"
guvnor restart
```

### Changing Ports
```bash
# Edit guvnor.yaml
apps:
  - name: api
    port: 8001        # Change from 8000 to 8001
    
guvnor validate      # Check config
guvnor restart       # Apply changes
```

### Adding Health Checks
```bash
# guvnor.yaml
apps:
  - name: api
    health_check:
      enabled: true
      path: /health    # Your health endpoint
      interval: 30s
      
guvnor restart
guvnor status        # Shows health status
```

## Team Workflows

### Onboarding New Developer
```bash
# New developer setup
git clone team-repo
cd team-repo
guvnor init          # Uses existing config
guvnor start         # Everything works

# No Docker, complex setup, or environment issues!
```

### Sharing Configuration
```bash
# Commit guvnor.yaml to git
git add guvnor.yaml
git commit -m "Add Guvnor configuration"

# Don't commit .env (secrets)
echo ".env" >> .gitignore
```

### Environment-Specific Configs
```bash
# Different configs for different environments
cp guvnor.yaml guvnor.dev.yaml
cp guvnor.yaml guvnor.prod.yaml

# Use specific config
guvnor start --config guvnor.dev.yaml
guvnor start --config guvnor.prod.yaml
```

## Migration Workflows

### From Docker Compose
```bash
# Old way
docker-compose up -d
docker-compose logs -f

# New way
guvnor init          # Auto-detects services
guvnor start         # Much faster startup
guvnor logs
```

### From Manual Process Management
```bash
# Old way
python api.py &
node frontend/server.js &
redis-server &

# New way
guvnor init          # Detects all processes
guvnor start         # Manages everything
# Automatic restarts, health checks, logging
```

### To Production (PM2/Systemd)
```bash
# Development
guvnor start

# Production: Install as service
sudo cp guvnor /usr/local/bin/
# Setup systemd service (see systemd.md)
sudo systemctl start guvnor
```

## Tips

- **Always run `guvnor validate`** before starting in production
- **Use `guvnor logs -f`** to follow logs in real-time
- **Keep .env files out of version control** (add to .gitignore)
- **Use different ports** for different projects
- **Test with `--staging`** before using production certificates