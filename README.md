# Stockyard Recipe

**Self-hosted recipe management with ingredient scaling**

Part of the [Stockyard](https://stockyard.dev) family of self-hosted tools.

## Quick Start

```bash
curl -fsSL https://stockyard.dev/tools/recipe/install.sh | sh
```

Or with Docker:

```bash
docker run -p 9805:9805 -v recipe_data:/data ghcr.io/stockyard-dev/stockyard-recipe
```

Open `http://localhost:9805` in your browser.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9805` | HTTP port |
| `DATA_DIR` | `./recipe-data` | SQLite database directory |
| `STOCKYARD_LICENSE_KEY` | *(empty)* | License key for unlimited use |

## Free vs Pro

| | Free | Pro |
|-|------|-----|
| Limits | 5 records | Unlimited |
| Price | Free | Included in bundle or $29.99/mo individual |

Get a license at [stockyard.dev](https://stockyard.dev).

## License

Apache 2.0
