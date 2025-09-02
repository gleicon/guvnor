# Getting Started

## New Project

```bash
mkdir my-app && cd my-app
echo 'console.log("Hello")' > server.js
echo '{"name": "my-app"}' > package.json

guvnor init
guvnor start
```

Visit `http://my-app.localhost:8080`

## Existing Project

```bash
cd my-project
guvnor init
guvnor start
```

Guvnor detects your app and generates config.

## Heroku App

```bash
# Existing Procfile:
# web: gunicorn app:app
# worker: celery -A app worker

guvnor init     # Uses Procfile
guvnor start
```

Web runs on `http://web.localhost:8080`, worker in background.

## Multiple Apps

Edit `guvnor.yaml`:

```yaml
apps:
  - name: web
    command: npm
    args: ["start"]
  - name: api  
    command: python
    args: ["api.py"]
```

Each app gets its own hostname.

## Production

```bash
guvnor start --domain myapp.com --email admin@myapp.com
```

Automatic HTTPS with Let's Encrypt.

## Daily Use

```bash
guvnor start    # Start apps
guvnor logs     # View logs  
guvnor restart  # Restart all
guvnor stop     # Stop all
```

## Config Priority

1. `guvnor.yaml` (primary)
2. `Procfile` (fallback)
3. Auto-detection

## Files

- `guvnor.yaml` - Main config
- `Procfile` - Heroku compatibility
- `.env` - Environment variables

## Docs

- [Config Reference](configuration.md)
- [Examples](examples.md)
- [Production](systemd.md)