# mairlist-feeder

Reads folder contents and creates a mAirList playlist for import

## Runtime Requirements

- Go, matching the version in `go.mod`
- `ffprobe` for audio metadata extraction
- A crawl folder organized as `YYYY/MM/DD`, for example `2026/05/18`
- Optional: a reachable mAirList Web Remote API when playlist appending or status polling is enabled
- Optional: a reachable calCMS endpoint when calCMS event enrichment is enabled

## Configuration

Configuration is read from `.env` by default. Use `-config.file <path>` to load a different file.

Important settings:

- `ROOT_FOLDER`: root folder containing date-based subfolders
- `EXPORT_FOLDER`: destination for generated `.tpi` playlists and exported HTML state
- `FFPROBE_PATH`: path to `ffprobe`
- `CRAWL_CYCLE_MIN`: crawl interval in minutes
- `EXPORT_MINUTE`: minute of each hour when playlist export runs
- `MAIRLIST_URL`, `MAIRLIST_USER`, `MAIRLIST_PASS`, `MAIRLIST_VERSION`: mAirList API settings
- `QUERY_CALCMS`, `CALCMS_URL`, `CALCMS_TEMPLATE`: calCMS integration
- `QUERY_MAIRLIST_STATUS`: enables background playback-status polling

## Web UI

The web UI exposes:

- `/`: runtime status
- `/filelist`: known files
- `/events`: cached calCMS event/file status
- `/actions`: manual crawl, export, clean, and save actions
- `/actions/:id`: status of a queued manual action
- `/logs`: in-memory logs
- `/metrics`: Prometheus metrics

## Development

Run the test suite with:

```sh
go test ./...
```

Manual actions are queued and executed serially. `POST /actions` returns `202 Accepted`
with a `status_url`; the web UI polls that URL until the action succeeds or fails.

Startup validates the configured crawl root, `ffprobe` executable, export directory,
and TLS files before accepting traffic.
