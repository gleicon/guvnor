# Running Guvnor as a systemd service

## Create systemd service file

```bash
sudo nano /etc/systemd/system/guvnor.service
```

```ini
[Unit]
Description=Guvnor Process Manager
After=network.target
Wants=network.target

[Service]
Type=exec
User=www-data
Group=www-data
WorkingDirectory=/opt/myapp
ExecStart=/usr/local/bin/guvnor start
ExecReload=/bin/kill -HUP $MAINPID
KillMode=mixed
TimeoutStopSec=30
Restart=always
RestartSec=5

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/myapp
ReadWritePaths=/var/lib/guvnor

# Environment
Environment=PATH=/usr/local/bin:/usr/bin:/bin
Environment=NODE_ENV=production

[Install]
WantedBy=multi-user.target
```

## Setup permissions

```bash
# Create guvnor user (optional)
sudo useradd -r -s /bin/false -d /opt/myapp guvnor

# Or use existing user
sudo chown -R www-data:www-data /opt/myapp
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