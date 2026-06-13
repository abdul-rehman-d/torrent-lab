# Torrent Panel

A small home-server torrent panel for qBittorrent. It adds magnet links or `.torrent` files, shows progress and ETA, and moves completed torrents into movie/TV folders.

The app is packaged as one Go server:

- `qbittorrent`: qBittorrent Web API and torrent client.
- `backend`: Go API that talks to qBittorrent Web API and serves the frontend.
- `frontend`: SvelteKit UI built to static files and served by Go.

## Behavior

- qBittorrent is expected at `http://qbittorrent:4000` from Docker by default.
- If qBittorrent is included in Compose, expose its Web UI at `http://localhost:4000`.
- Magnet links and torrent files are added to qBittorrent with `savepath=/data/torrents/downloads`.
- The UI and API are served from the same port. API routes live under `/api`.
- The UI polls the API every 2 seconds for progress and ETA.
- Completed torrents are moved through qBittorrent's `setLocation` API, so qBittorrent keeps seeding from the new library path.
- The original files under `/data/torrents/downloads` are removed by qBittorrent's move operation.
- Manual remove removes the torrent from qBittorrent without deleting files.
- TV detection uses `S\d\d` or `S\d\dE\d\d` in filenames.

## Paths

The default Compose setup uses local folders under `./data` for testing.

```txt
./data/media/movies
./data/media/tv
./data/torrents/incomplete
./data/torrents/downloads
./data/qbittorrent/config
```

Inside Docker, those folders are mounted as:

```txt
/data/media/movies
/data/media/tv
/data/torrents/incomplete
/data/torrents/downloads
```

TV torrents are moved to:

```txt
/data/media/tv/Show Name/Season 01/Episode.File.S01E01.mkv
```

Movie torrents are moved to:

```txt
/data/media/movies/Movie Title/Movie.Title.1080p.mkv
```

## Run

Create `.env` from `.env.example` and update qBittorrent credentials if needed.

```sh
docker compose -f testing-docker-compose.yml up --build
```

Open:

```txt
http://localhost:8080
```

qBittorrent Web UI:

```txt
http://localhost:4000
```

The qBittorrent config is stored at `./data/qbittorrent/config` by default.

On first boot, qBittorrent may generate a temporary admin password in its container logs. Log in to the Web UI and set the password to match `QBIT_PASSWORD`, or update `QBIT_PASSWORD` to match what you configure in qBittorrent.

## Environment

```env
QBIT_USERNAME=admin
QBIT_PASSWORD=adminadmin
API_PORT=8080
PUID=1000
PGID=1000
TZ=Etc/UTC
TORRENTING_PORT=6881
```

## Development

Backend:

```sh
cd backend
go test ./...
WEB_DIR=../frontend/build go run ./cmd/server
```

Frontend:

```sh
cd frontend
npm install
npm run build
```
