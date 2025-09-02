# How to run your Rust app with Guvnor

## Setup

```bash
cd my-rust-app
cargo build --release
guvnor init
```

## Configuration

```yaml
# guvnor.yaml
server:
  http_port: 8080
  https_port: 8443
  log_level: info

apps:
  - name: rust-app
    hostname: localhost
    port: 3000
    command: ./target/release/myapp
    working_dir: /path/to/your/rust-app
    environment:
      RUST_ENV: "production"
      PORT: "3000"
    health_check:
      enabled: true
      path: /health
      interval: 30s
    restart_policy:
      enabled: true
      max_retries: 5
      backoff: 3s

tls:
  enabled: false
  auto_cert: false
  cert_dir: ./certs
```

## Usage

```bash
# Production deployment
cargo build --release          # Build first
guvnor start                    # Access: http://localhost:8080/
guvnor logs                     # View logs
guvnor status                   # Check status
guvnor stop                     # Stop app

# Development
guvnor start                    # If using dev config below
```

## Alternative Configurations

### Development Mode
```yaml
apps:
  - name: rust-dev
    command: cargo
    args: ["run"]
    environment:
      RUST_ENV: "development"
```

### With Cargo Watch
```yaml
apps:
  - name: rust-watch
    command: cargo
    args: ["watch", "-x", "run"]
    environment:
      RUST_ENV: "development"
```

### Debug Build
```yaml
apps:
  - name: rust-debug
    command: ./target/debug/myapp
    environment:
      RUST_ENV: "development"
      RUST_LOG: "debug"
```

## Build Commands

```bash
# Release build
cargo build --release

# Debug build
cargo build

# Cross-compile for Linux
cargo build --release --target x86_64-unknown-linux-gnu

# Static binary with musl
cargo build --release --target x86_64-unknown-linux-musl
```

## Required Cargo.toml

```toml
[package]
name = "myapp"
version = "0.1.0"
edition = "2021"

[[bin]]
name = "myapp"
path = "src/main.rs"

[dependencies]
# Your dependencies

[profile.release]
lto = true
codegen-units = 1
panic = "abort"
```