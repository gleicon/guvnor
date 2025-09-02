# How to run your PHP app with Guvnor

## Setup

```bash
cd my-php-app
composer install --no-dev --optimize-autoloader
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
  - name: php-app
    hostname: localhost
    port: 3000
    command: php
    args:
      - "-S"
      - "0.0.0.0:3000"
      - "-t"
      - "public"
    working_dir: /path/to/your/php-app
    environment:
      APP_ENV: "production"
      PHP_CLI_SERVER_WORKERS: "4"
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
composer install --no-dev       # Install dependencies
guvnor start                    # Access: http://localhost:8080/
guvnor logs                     # View logs
guvnor status                   # Check status
guvnor stop                     # Stop app

# Development
guvnor start                    # If using dev config below
```

## Alternative Configurations

### Laravel Application
```yaml
apps:
  - name: laravel-app
    command: php
    args: ["artisan", "serve", "--host=0.0.0.0", "--port=3000"]
    environment:
      APP_ENV: "production"
      APP_KEY: "your-app-key"
```

### With PHP-FPM + Nginx
```yaml
apps:
  - name: php-fpm
    command: php-fpm
    args: ["--nodaemonize", "--fpm-config", "/etc/php-fpm.conf"]
    port: 9000

  - name: nginx-php
    hostname: localhost
    port: 3000
    command: nginx
    args: ["-g", "daemon off;"]
```

### Symfony Application
```yaml
apps:
  - name: symfony-app
    command: php
    args: ["-S", "0.0.0.0:3000", "-t", "public"]
    environment:
      APP_ENV: "prod"
      APP_SECRET: "your-app-secret"
```

### Development Mode
```yaml
apps:
  - name: php-dev
    command: php
    args: ["-S", "0.0.0.0:3000", "-t", "public"]
    environment:
      APP_ENV: "development"
      APP_DEBUG: "true"
```

## Required composer.json

```json
{
    "name": "my/php-app",
    "type": "project",
    "require": {
        "php": ">=8.1"
    },
    "autoload": {
        "psr-4": {
            "App\\": "src/"
        }
    },
    "config": {
        "optimize-autoloader": true
    }
}
```

## PHP Requirements

- PHP 8.1 or higher
- Extensions: mbstring, openssl, pdo, tokenizer
- Composer for dependency management