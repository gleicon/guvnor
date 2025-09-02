# How to run your React app with Guvnor

## Setup

```bash
cd my-react-app
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
  - name: react-app
    hostname: localhost
    port: 3000
    command: npx
    args:
      - "serve"
      - "-s"
      - "build"
      - "-l"
      - "3000"
    working_dir: /path/to/your/react-app
    environment:
      NODE_ENV: "production"
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
# Production deployment
npm run build                   # Build first
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
  - name: react-dev
    command: npm
    args: ["start"]
    environment:
      NODE_ENV: "development"
      BROWSER: "none"
```

### Using Express Server
```yaml
apps:
  - name: react-express
    command: node
    args: ["server.js"]
    environment:
      NODE_ENV: "production"
```

### Using Nginx-style Static
```yaml
apps:
  - name: react-static
    command: python3
    args: ["-m", "http.server", "3000"]
    working_dir: /path/to/your/react-app/build
```

## Required package.json scripts

```json
{
  "scripts": {
    "start": "react-scripts start",
    "build": "react-scripts build"
  },
  "dependencies": {
    "serve": "^14.0.0"
  }
}
```

## Install serve globally

```bash
npm install -g serve
```