<div align="center">
  <img src="web/public/logo.png" alt="eBook Reader" width="128" />

  # eBook Reader

  A self-hosted EPUB reader: scan a folder of books, read in the browser, match metadata, and track progress. Single user, no login. Bind it to your LAN, or put a reverse proxy in front if it is on a public IP.

</div>

The app never writes to your EPUB files. Covers and the SQLite database live in a separate data directory.

## Docker

Published images: [`ghcr.io/jdbnet/ebook-reader`](https://github.com/jdbnet/ebook-reader/pkgs/container/ebook-reader).

Copy `docker-compose.yml` from this repo, or use:

```yaml
services:
  ebook-reader:
    image: ghcr.io/jdbnet/ebook-reader:latest
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
      - ebook-reader-data:/data
    restart: unless-stopped

volumes:
  ebook-reader-data:
```

Replace `/path/to/ebooks` with the folder that holds your `.epub` files, then:

```bash
docker compose up -d
```

Open `http://localhost:8080`. Pin a version with `ghcr.io/jdbnet/ebook-reader:v1.0.0` instead of `:latest`.

The compose file in this repo uses the same image. Set `LIBRARY_HOST_PATH` if you do not want `./library`:

```bash
LIBRARY_HOST_PATH=/mnt/ebooks docker compose up -d
```

To build locally instead of pulling: `docker compose up -d --build`.

## Binary

Download `ebook-reader-linux-amd64` from [GitHub Releases](https://github.com/jdbnet/ebook-reader/releases/latest).

```bash
chmod +x ebook-reader-linux-amd64
./ebook-reader-linux-amd64 --library /path/to/ebooks --data ./data
```

Then open `http://localhost:8080`.

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
./ebook-reader-linux-amd64 --version
```
