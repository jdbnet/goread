<div align="center">
  <img src="web/public/logo.png" alt="GoRead" width="128" />

  # GoRead

  A self-hosted EPUB reader: scan a folder of books, read in the browser, match metadata, and track progress. Single user. Login is optional: set a username and password in Settings when you want one.

</div>

The app never writes to your EPUB files. Covers and the SQLite database live in a separate data directory.

## Authentication

The app starts open. Anyone who can reach it can use it until you turn login on.

To require a login, open **Settings**, choose a username and password (at least 8 characters), and save. After that, every visit needs those credentials. The password is hashed in the SQLite database; it is not stored in plaintext. Sessions last 30 days.

You can change the username or password later from Settings (current password required). To go back to an open app, enter the current password and turn login off.

## Docker

Published images: [`ghcr.io/jdbnet/goread`](https://github.com/jdbnet/goread/pkgs/container/goread).

Copy `docker-compose.yml` from this repo, or use:

```yaml
services:
  goread:
    image: ghcr.io/jdbnet/goread:latest
    ports:
      - "8080:8080"
    environment:
      LIBRARY_PATH: /library
      DATA_DIR: /data
      TZ: Europe/London
      SCAN_INTERVAL: 5m
    volumes:
      # Your existing EPUB folder. Read-only: the app never rewrites books.
      - /path/to/ebooks:/library:ro
      # SQLite database and covers.
      - goread-data:/data
    restart: unless-stopped

volumes:
  goread-data:
```

Replace `/path/to/ebooks` with the folder that holds your `.epub` files, then:

```bash
docker compose up -d
```

Open `http://localhost:8080`. If you have enabled login in Settings, you will be asked to sign in. Pin a version with `ghcr.io/jdbnet/goread:v1.0.0` instead of `:latest`.

The compose file in this repo uses the same image. Set `LIBRARY_HOST_PATH` if you do not want `./library`:

```bash
LIBRARY_HOST_PATH=/mnt/ebooks docker compose up -d
```

To build locally instead of pulling: `docker compose up -d --build`.

## Binary

Download `goread-linux-amd64` from [GitHub Releases](https://github.com/jdbnet/goread/releases/latest).

```bash
chmod +x goread-linux-amd64
./goread-linux-amd64 --library /path/to/ebooks --data ./data
```

Then open `http://localhost:8080`. Same as Docker: the library is reachable immediately until you set a login in Settings.

## Configuration

Flags and environment variables are equivalent. Library path is required.

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `--library` | `LIBRARY_PATH` | (required) | Folder of EPUB files |
| `--data` | `DATA_DIR` | `./data` | SQLite DB and covers |
| `--listen` | `LISTEN` | `:8080` | HTTP bind address |
| `--tz` | `TZ` | `UTC` | Timezone for daily stats and streaks |
| `--scan-interval` | `SCAN_INTERVAL` | `5m` | Rescan interval; `0` disables |
| `--fsnotify` | `FSNOTIFY` | `true` | Watch the library for file changes |

```bash
./goread-linux-amd64 --version
```
