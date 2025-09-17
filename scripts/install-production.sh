#!/bin/bash
# Production Installation Script for Guvnor
# Installs and configures Guvnor as a systemd service with security best practices

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GUVNOR_USER="guvnor"
INSTALL_DIR="/opt/myapp"
CERT_DIR="/var/lib/guvnor/certs"
LOG_DIR="/var/log/guvnor"
SERVICE_FILE="/etc/systemd/system/guvnor.service"

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

detect_architecture() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "x86_64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

detect_os() {
    local os=$(uname -s)
    case $os in
        Linux)
            echo "Linux"
            ;;
        Darwin)
            echo "Darwin"
            ;;
        *)
            print_error "Unsupported OS: $os"
            exit 1
            ;;
    esac
}

install_guvnor() {
    print_status "Installing Guvnor binary..."
    
    local os=$(detect_os)
    local arch=$(detect_architecture)
    local binary_name="guvnor-${os}-${arch}"
    local download_url="https://github.com/gleicon/guvnor/releases/latest/download/${binary_name}"
    
    # Download binary
    print_status "Downloading from: $download_url"
    curl -sSL "$download_url" -o /tmp/guvnor
    
    # Install binary
    mv /tmp/guvnor /usr/local/bin/guvnor
    chmod +x /usr/local/bin/guvnor
    
    # Verify installation
    if /usr/local/bin/guvnor --version; then
        print_success "Guvnor binary installed successfully"
    else
        print_error "Failed to install Guvnor binary"
        exit 1
    fi
}

create_user() {
    print_status "Creating system user: $GUVNOR_USER"
    
    if id "$GUVNOR_USER" &>/dev/null; then
        print_warning "User $GUVNOR_USER already exists"
    else
        useradd -r -s /bin/false -d "$INSTALL_DIR" -c "Guvnor Process Manager" "$GUVNOR_USER"
        print_success "Created user: $GUVNOR_USER"
    fi
}

create_directories() {
    print_status "Creating directories..."
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CERT_DIR"
    mkdir -p "$LOG_DIR"
    
    # Set ownership
    chown -R "$GUVNOR_USER:$GUVNOR_USER" "$INSTALL_DIR" "$CERT_DIR" "$LOG_DIR"
    
    # Set permissions
    chmod 755 "$INSTALL_DIR"
    chmod 700 "$CERT_DIR"
    chmod 755 "$LOG_DIR"
    
    print_success "Directories created and configured"
}

create_systemd_service() {
    print_status "Creating systemd service file..."
    
    cat > "$SERVICE_FILE" << 'EOF'
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

# Enhanced startup with configuration validation
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

# Security hardening
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
EOF

    print_success "Systemd service file created"
}

create_sample_config() {
    print_status "Creating sample configuration..."
    
    cat > "$INSTALL_DIR/guvnor.yaml.example" << 'EOF'
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
    hostname: example.com  # Change this to your domain
    port: 3000
    command: node
    args: ["dist/server.js"]
    environment:
      NODE_ENV: production
      PORT: "3000"
    tls:
      enabled: true
      auto_cert: true
      email: ops@example.com  # Change this to your email
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
    - example.com  # Change this to your domain
EOF

    chown "$GUVNOR_USER:$GUVNOR_USER" "$INSTALL_DIR/guvnor.yaml.example"
    print_success "Sample configuration created at $INSTALL_DIR/guvnor.yaml.example"
}

create_health_check() {
    print_status "Creating health check script..."
    
    cat > "$INSTALL_DIR/health-check.sh" << 'EOF'
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
EOF

    chmod +x "$INSTALL_DIR/health-check.sh"
    chown "$GUVNOR_USER:$GUVNOR_USER" "$INSTALL_DIR/health-check.sh"
    print_success "Health check script created"
}

setup_firewall() {
    print_status "Configuring firewall..."
    
    if command -v ufw >/dev/null 2>&1; then
        ufw allow 80/tcp
        ufw allow 443/tcp
        print_success "UFW firewall configured (ports 80, 443 opened)"
    elif command -v firewall-cmd >/dev/null 2>&1; then
        firewall-cmd --permanent --add-service=http
        firewall-cmd --permanent --add-service=https
        firewall-cmd --reload
        print_success "Firewalld configured (HTTP, HTTPS services enabled)"
    else
        print_warning "No supported firewall found. Please manually open ports 80 and 443"
    fi
}

configure_systemd() {
    print_status "Configuring systemd..."
    
    systemctl daemon-reload
    systemctl enable guvnor
    
    print_success "Guvnor service enabled"
    print_warning "Service is not started yet. Configure guvnor.yaml first."
}

print_next_steps() {
    print_success "Guvnor installation completed!"
    echo
    echo -e "${BLUE}Next steps:${NC}"
    echo "1. Copy your application to $INSTALL_DIR"
    echo "2. Configure $INSTALL_DIR/guvnor.yaml (see example file)"
    echo "3. Update domain and email in the configuration"
    echo "4. Start the service: sudo systemctl start guvnor"
    echo "5. Check status: sudo systemctl status guvnor"
    echo "6. View logs: sudo journalctl -u guvnor -f"
    echo
    echo -e "${BLUE}Useful commands:${NC}"
    echo "• Validate config: sudo -u $GUVNOR_USER guvnor validate --config $INSTALL_DIR/guvnor.yaml"
    echo "• Health check: $INSTALL_DIR/health-check.sh"
    echo "• Service logs: sudo journalctl -u guvnor"
    echo "• Certificate info: sudo -u $GUVNOR_USER guvnor cert info --config $INSTALL_DIR/guvnor.yaml"
}

main() {
    echo -e "${GREEN}Guvnor Production Installation Script${NC}"
    echo "===================================="
    echo
    
    check_root
    install_guvnor
    create_user
    create_directories
    create_systemd_service
    create_sample_config
    create_health_check
    setup_firewall
    configure_systemd
    
    print_next_steps
}

# Run main function
main "$@"