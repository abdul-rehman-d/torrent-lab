# Torrent Panel

Torrent Panel is a lightweight web UI for qBittorrent in a home lab. It adds magnet links or `.torrent` files, shows torrent progress, and moves completed downloads into movie and TV library folders.

The container includes the Go API and the built Svelte frontend. qBittorrent runs as a separate container or service.

## Quick Start

Create a `docker-compose.yml`:

```yaml
services:
  qbittorrent:
    image: lscr.io/linuxserver/qbittorrent:latest
    container_name: qbittorrent
    environment:
      PUID: 1000
      PGID: 1000
      TZ: Etc/UTC
      WEBUI_PORT: 4000
      TORRENTING_PORT: 6881
    ports:
      - "4000:4000"
      - "6881:6881"
      - "6881:6881/udp"
    volumes:
      - ./data/qbittorrent/config:/config
      - ./data/media/movies:/media/movies
      - ./data/media/tv:/media/tv
      - ./data/torrents/incomplete:/torrents/incomplete
      - ./data/torrents/downloads:/torrents/downloads
    restart: unless-stopped

  torrent-panel:
    image: ghcr.io/OWNER/REPO:latest
    container_name: torrent-panel
    environment:
      CONFIG_DIR: /config
      MOVIES_PATH: /media/movies
      TV_PATH: /media/tv
      INCOMPLETE_PATH: /torrents/incomplete
      DOWNLOADS_PATH: /torrents/downloads
      PORT: 8080
    ports:
      - "8080:8080"
    volumes:
      - ./data/config:/config
      - ./data/media/movies:/media/movies
      - ./data/media/tv:/media/tv
      - ./data/torrents/incomplete:/torrents/incomplete
      - ./data/torrents/downloads:/torrents/downloads
    depends_on:
      - qbittorrent
    restart: unless-stopped
```

Replace `ghcr.io/OWNER/REPO:latest` with this repository's GitHub Container Registry image, for example `ghcr.io/your-name/torrent:latest`.

Start the stack:

```sh
docker compose up -d
```

Open Torrent Panel at `http://localhost:8080`.

Open qBittorrent at `http://localhost:4000`.

On first boot, qBittorrent may print a temporary admin password in its container logs. Log in to qBittorrent, set the credentials you want, then configure Torrent Panel to use:

```txt
URL: http://qbittorrent:4000
Username: your-qbittorrent-username
Password: your-qbittorrent-password
```

## Image Tags

Images are published to GitHub Container Registry by the Docker Image workflow.

```txt
ghcr.io/OWNER/REPO:latest
ghcr.io/OWNER/REPO:master
ghcr.io/OWNER/REPO:sha-<commit>
ghcr.io/OWNER/REPO:<version>
```

`latest` is published from the default branch. Version tags are published when a Git tag like `v1.2.3` is pushed.

## Configuration

Torrent Panel stores qBittorrent connection settings in `qbit.json` under `CONFIG_DIR`. In the Compose example, that file is stored at `./data/config/qbit.json`.

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port inside the Torrent Panel container. |
| `CONFIG_DIR` | `./config` | Directory where Torrent Panel stores `qbit.json`. Use `/config` in Docker. |
| `WEB_DIR` | `./web` | Directory containing the built frontend. The published image sets this to `/web`. |
| `MOVIES_PATH` | `/data/media/movies` | Movie library path visible inside the Torrent Panel container. |
| `TV_PATH` | `/data/media/tv` | TV library path visible inside the Torrent Panel container. |
| `INCOMPLETE_PATH` | `/data/torrents/incomplete` | Incomplete download path. Keep this aligned with qBittorrent. |
| `DOWNLOADS_PATH` | `/data/torrents/downloads` | Completed download path. Keep this aligned with qBittorrent. |

The path variables must point to paths that both Torrent Panel and qBittorrent can access. The host paths may be different, but the mounted container paths should match the values configured in Torrent Panel and qBittorrent.

## qBittorrent Settings

For the Compose example above, set qBittorrent's paths to:

| Setting | Value |
| --- | --- |
| Web UI port | `4000` |
| Incomplete downloads | `/torrents/incomplete` |
| Default save path | `/torrents/downloads` |

When Torrent Panel and qBittorrent are in the same Compose file, use `http://qbittorrent:4000` as the qBittorrent URL. Do not use `localhost` from inside the Torrent Panel container.

## Behavior

Torrent Panel adds magnet links and torrent files to qBittorrent using the configured downloads path. The UI and API are served from the same port, with API routes under `/api`.

Completed torrents are moved through qBittorrent's `setLocation` API, so qBittorrent keeps seeding from the new library path. Manual removal removes the torrent from qBittorrent without deleting files.

TV detection uses `S01` or `S01E01` style filename markers. TV files are moved under `TV_PATH`, and movie files are moved under `MOVIES_PATH`.

## Local Testing

This repository includes a local Compose file that builds the image from source:

```sh
docker compose -f testing-docker-compose.yml up --build
```

The test stack stores data under `./data`.

## Development

Build the frontend before running the backend locally:

```sh
cd frontend
npm install
npm run build
```

Run the backend:

```sh
cd backend
go test ./...
WEB_DIR=../frontend/build CONFIG_DIR=../data/config go run ./cmd/server
```

Useful API endpoints:

```txt
GET /api/health
GET /api/config
PUT /api/config/qbit
```
