# Stockyard Tally

**Event tracking — script tag, custom events, funnels, no cookies, no GDPR problem**

Part of the [Stockyard](https://stockyard.dev) family of self-hosted developer tools.

## Quick Start

```bash
docker run -p 9110:9110 -v tally_data:/data ghcr.io/stockyard-dev/stockyard-tally
```

Or with docker-compose:

```bash
docker-compose up -d
```

Open `http://localhost:9110` in your browser.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9110` | HTTP port |
| `DATA_DIR` | `./data` | SQLite database directory |
| `TALLY_LICENSE_KEY` | *(empty)* | Pro license key |

## Free vs Pro

| | Free | Pro |
|-|------|-----|
| Limits | 1 project, 10k events/mo | Unlimited projects and events |
| Price | Free | $4.99/mo |

Get a Pro license at [stockyard.dev/tools/](https://stockyard.dev/tools/).

## Category

Developer Tools

## License

Apache 2.0
