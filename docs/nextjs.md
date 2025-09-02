# How to run your Next.js app with Guvnor

## Setup

```bash
cd my-nextjs-app
npm install
npm run build
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
  - name: nextjs-app
    hostname: localhost
    port: 3000
    command: npm
    args:
      - "start"
    working_dir: /path/to/your/nextjs-app
    environment:
      NODE_ENV: "production"
      PORT: "3000"
    health_check:
      enabled: true
      path: /
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
# Development
guvnor start                    # Access: http://localhost:8080/
guvnor logs                     # View logs
guvnor status                   # Check status
guvnor stop                     # Stop app

# Production deployment
npm run build                   # Build first
guvnor start                    # Then start
```

## Alternative Configurations

### Development Mode
```yaml
apps:
  - name: nextjs-dev
    command: npm
    args: ["run", "dev"]
    environment:
      NODE_ENV: "development"
```

### Direct Node
```yaml
apps:
  - name: nextjs-app
    command: node
    args: [".next/standalone/server.js"]
    environment:
      NODE_ENV: "production"
```

### Static Export
```yaml
apps:
  - name: nextjs-static
    command: npx
    args: ["serve", "out", "-p", "3000"]
```

## Required package.json scripts

```json
{
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start"
  }
}
```