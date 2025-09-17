# Production Deployment with systemd

Complete guide for deploying Guvnor as a production systemd service with security best practices.

## Quick Production Setup

```bash
# 1. Install Guvnor
curl -sSL https://github.com/gleicon/guvnor/releases/latest/download/guvnor-$(uname -s)-$(uname -m) -o guvnor
sudo mv guvnor /usr/local/bin/
sudo chmod +x /usr/local/bin/guvnor

# 2. Create deployment directory
sudo mkdir -p /opt/myapp
sudo mkdir -p /var/lib/guvnor/certs
sudo mkdir -p /var/log/guvnor

# 3. Create system user
sudo useradd -r -s /bin/false -d /opt/myapp -c "Guvnor Process Manager" guvnor

# 4. Set permissions
sudo chown -R guvnor:guvnor /opt/myapp /var/lib/guvnor /var/log/guvnor
```

## Enhanced systemd Service File

Create `/etc/systemd/system/guvnor.service`:

```ini
[Unit]
Description=Guvnor Process Manager and Reverse Proxy
Documentation=https://github.com/gleicon/guvnor
After=network-online.target
Wants=network-online.target
Requires=network.target

[Service]
Type=exec
User=guvnor
Group=guvnor
WorkingDirectory=/opt/myapp

# 🆕 Enhanced startup with configuration validation
ExecStartPre=/usr/local/bin/guvnor validate
ExecStart=/usr/local/bin/guvnor start --config /opt/myapp/guvnor.yaml
ExecReload=/usr/local/bin/guvnor restart
ExecStop=/usr/local/bin/guvnor stop

# Process management
KillMode=mixed
KillSignal=SIGTERM
TimeoutStartSec=60
TimeoutStopSec=30
TimeoutReloadSec=30
Restart=always
RestartSec=10
StartLimitInterval=300
StartLimitBurst=5

# Security hardening 🔒
NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictSUIDSGID=true
LockPersonality=true
MemoryDenyWriteExecute=false
SystemCallArchitectures=native

# File system access
ReadWritePaths=/opt/myapp
ReadWritePaths=/var/lib/guvnor
ReadWritePaths=/var/log/guvnor
ReadOnlyPaths=/usr/local/bin/guvnor

# Capabilities (for binding to ports 80/443)
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

# Environment variables
Environment=PATH=/usr/local/bin:/usr/bin:/bin
Environment=NODE_ENV=production
Environment=GUVNOR_LOG_DIR=/var/log/guvnor
Environment=GUVNOR_CERT_DIR=/var/lib/guvnor/certs

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

## Enhanced Production Configuration

Create `/opt/myapp/guvnor.yaml`:

```yaml
# Production Guvnor Configuration
server:
  http_port: 80
  https_port: 443
  log_level: warn
  enable_tracking: true
  tracking_header: "X-REQUEST-ID"
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 30s

apps:
  - name: production-app
    hostname: myapp.com
    port: 3000
    command: node
    args: ["dist/server.js"]
    environment:
      NODE_ENV: production
      PORT: "3000"
    tls:
      enabled: true
      auto_cert: true
      email: ops@myapp.com
      staging: false
    health_check:
      enabled: true
      path: /health
      interval: 15s
      timeout: 5s
      retries: 3
    restart_policy:
      enabled: true
      max_retries: 10
      backoff: 10s

tls:
  enabled: true
  auto_cert: true
  cert_dir: /var/lib/guvnor/certs
  force_https: true
  certificate_headers: true
  domains:
    - myapp.com
```

## Service Management Commands

```bash
# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable guvnor
sudo systemctl start guvnor

# Check status
sudo systemctl status guvnor
sudo journalctl -u guvnor -f

# Restart and reload
sudo systemctl restart guvnor
sudo systemctl reload guvnor

# Stop service
sudo systemctl stop guvnor

# View logs
sudo journalctl -u guvnor --since "1 hour ago"
sudo journalctl -u guvnor -f --lines=100
```

## Log Management with journald

Configure log retention in `/etc/systemd/journald.conf`:

```ini
[Journal]
SystemMaxUse=1G
SystemMaxFileSize=100M
SystemMaxFiles=10
MaxRetentionSec=2week
```

## Monitoring and Alerts

### Basic Health Check Script

Create `/opt/myapp/health-check.sh`:

```bash
#!/bin/bash
# Basic health check for Guvnor service

set -e

# Check if service is running
if ! systemctl is-active --quiet guvnor; then
    echo "ERROR: Guvnor service is not running"
    exit 1
fi

# Check HTTP endpoint
if ! curl -f -s http://localhost/health > /dev/null; then
    echo "ERROR: Health endpoint not responding"
    exit 1
fi

# Check process count
PROCESS_COUNT=$(guvnor status | grep -c "running" || true)
if [ "$PROCESS_COUNT" -eq 0 ]; then
    echo "ERROR: No processes running"
    exit 1
fi

echo "OK: Guvnor is healthy ($PROCESS_COUNT processes running)"
```

### Systemd Timer for Health Checks

Create `/etc/systemd/system/guvnor-health.service`:

```ini
[Unit]
Description=Guvnor Health Check
After=guvnor.service
Requires=guvnor.service

[Service]
Type=oneshot
User=guvnor
ExecStart=/opt/myapp/health-check.sh
```

Create `/etc/systemd/system/guvnor-health.timer`:

```ini
[Unit]
Description=Run Guvnor Health Check every 5 minutes
Requires=guvnor-health.service

[Timer]
OnCalendar=*:0/5
Persistent=true

[Install]
WantedBy=timers.target
```

Enable the timer:
```bash
sudo chmod +x /opt/myapp/health-check.sh
sudo systemctl enable guvnor-health.timer
sudo systemctl start guvnor-health.timer
```
sudo chmod +x /usr/local/bin/guvnor

# Create directories
sudo mkdir -p /var/lib/guvnor/certs
sudo chown -R www-data:www-data /var/lib/guvnor
```

## Enable and start service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service (start on boot)
sudo systemctl enable guvnor

# Start service now
sudo systemctl start guvnor

# Check status
sudo systemctl status guvnor
```

## Service management commands

```bash
# Start service
sudo systemctl start guvnor

# Stop service  
sudo systemctl stop guvnor

# Restart service
sudo systemctl restart guvnor

# Reload configuration
sudo systemctl reload guvnor

# View logs
sudo journalctl -u guvnor -f

# View recent logs
sudo journalctl -u guvnor --since "1 hour ago"

# Enable/disable auto-start
sudo systemctl enable guvnor
sudo systemctl disable guvnor
```

## Production configuration

```ini
[Unit]
Description=Guvnor Process Manager
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=60
StartLimitBurst=3

[Service]
Type=exec
User=guvnor
Group=guvnor
WorkingDirectory=/opt/myapp
ExecStart=/usr/local/bin/guvnor start --config /etc/guvnor/guvnor.yaml
ExecReload=/bin/kill -HUP $MAINPID
KillMode=mixed
TimeoutStartSec=60
TimeoutStopSec=30
Restart=always
RestartSec=10

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=guvnor

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
ReadWritePaths=/opt/myapp
ReadWritePaths=/var/lib/guvnor
ReadWritePaths=/var/log/guvnor

# Capabilities
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

# Environment
Environment=PATH=/usr/local/bin:/usr/bin:/bin
Environment=HOME=/opt/myapp
EnvironmentFile=-/etc/guvnor/environment

[Install]
WantedBy=multi-user.target
```

## Environment file

```bash
# /etc/guvnor/environment
NODE_ENV=production
GO_ENV=production
RUST_ENV=production
APP_ENV=production
```

## Log rotation

```bash
# /etc/logrotate.d/guvnor
/var/log/guvnor/*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 guvnor guvnor
    postrotate
        systemctl reload guvnor
    endscript
}
```

## Troubleshooting

```bash
# Check service status
sudo systemctl status guvnor

# View detailed logs
sudo journalctl -u guvnor -n 50

# Check configuration
sudo systemd-analyze verify /etc/systemd/system/guvnor.service

# Test service file
sudo systemctl daemon-reload
sudo systemctl start guvnor
sudo systemctl status guvnor
```

## Multiple instances

```bash
# For multiple guvnor instances
sudo cp /etc/systemd/system/guvnor.service /etc/systemd/system/guvnor-app1.service
sudo cp /etc/systemd/system/guvnor.service /etc/systemd/system/guvnor-app2.service

# Edit each service file with different:
# - WorkingDirectory
# - User/Group  
# - ExecStart parameters
```