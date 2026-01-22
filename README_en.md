<div align="center">

# <img src="docs/static/logo.png" width="45" align="center"> Save Any Bot

**English** | [简体中文](./README.md)

> **Save Any Telegram File to Anywhere 📂. This fork adds chunked upload for OpenList/WebDAV to bypass Cloudflare upload limits.**

[![Release Date](https://img.shields.io/github/release-date/Haruka041/SaveAny-Bot?label=release)](https://github.com/Haruka041/SaveAny-Bot/releases)
[![tag](https://img.shields.io/github/v/tag/Haruka041/SaveAny-Bot.svg)](https://github.com/Haruka041/SaveAny-Bot/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/Haruka041/SaveAny-Bot/build-release.yml)](https://github.com/Haruka041/SaveAny-Bot/actions/workflows/build-release.yml)
[![Stars](https://img.shields.io/github/stars/Haruka041/SaveAny-Bot?style=flat)](https://github.com/Haruka041/SaveAny-Bot/stargazers)
[![Downloads](https://img.shields.io/github/downloads/Haruka041/SaveAny-Bot/total)](https://github.com/Haruka041/SaveAny-Bot/releases)
[![Issues](https://img.shields.io/github/issues/Haruka041/SaveAny-Bot)](https://github.com/Haruka041/SaveAny-Bot/issues)
[![Pull Requests](https://img.shields.io/github/issues-pr/Haruka041/SaveAny-Bot?label=pr)](https://github.com/Haruka041/SaveAny-Bot/pulls)
[![License](https://img.shields.io/github/license/Haruka041/SaveAny-Bot)](./LICENSE)

</div>

## Overview

SaveAny-Bot is a Telegram bot that saves files/messages from Telegram and various websites to multiple storage backends. This fork adds a chunked upload receiver to reliably upload large files to OpenList (local storage) without hitting Cloudflare's 100MB request body limit.

## Core Features

- Support documents / videos / photos / stickers… and even [Telegraph](https://telegra.ph/)
- Bypass "restrict saving content" media
- Batch download and streaming transfer
- Multi-user support with storage rules
- Watch specified chats and auto-save messages, with filters
- Transfer files between different storage backends
- Integrate with yt-dlp to download and save media from 1000+ websites
- Aria2 integration to download files from URLs/magnets and save to storages
- Write JS parser plugins to save files from almost any website
- Storage backends:
  - Alist
  - S3
  - WebDAV
  - Local filesystem
  - Telegram (re-upload to specified chats)

## Fork Features (Chunked Upload for OpenList)

- WebDAV storage can be routed to a chunked receiver via `receiver_url`.
- Resumable uploads using server-side offset checks.
- Staging to final directory with atomic move on completion.
- Upload manifests and append-only log for tracking.
- Automatic cleanup of stale staging files.

## Architecture

1. Bot uploads file in chunks to the receiver (`/upload_chunk`).
2. Receiver writes chunks to staging and records progress.
3. Bot calls `/complete`, receiver moves the file to OpenList local storage.
4. OpenList reads from local storage path, file becomes visible.

## Deployment

### 1) Receiver (Docker)

```bash
cd file-receiver
docker compose up -d --build
```

#### Receiver directory layout (example)

- `STAGING_DIR`: staging directory (container: `/data/staging`)
- `MANIFEST_DIR`: manifest directory (container: `/data/manifests`)
- `FINAL_DIR`: OpenList local storage (container: `/data/final`)

#### Docker compose example (sanitized)

```yaml
version: "3.8"

services:
  file-receiver:
    container_name: file-receiver
    build: .
    ports:
      - "8080:8080"
    environment:
      - FINAL_DIR=/data/final
      - STAGING_DIR=/data/staging
      - MANIFEST_DIR=/data/manifests
      - LOG_PATH=/data/manifests/uploads.log
      - STAGING_TTL_HOURS=48
    volumes:
      - ./staging:/data/staging
      - ./manifests:/data/manifests
      - /path/to/openlist/storage:/data/final
    restart: always
```

### 2) Receiver (systemd)

```bash
sudo cp file-receiver.service /etc/systemd/system/file-receiver.service
sudo systemctl daemon-reload
sudo systemctl enable --now file-receiver
```

#### systemd example (sanitized)

```ini
[Unit]
Description=File Receiver Service
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/file-receiver
Environment=STAGING_DIR=/opt/file-receiver/staging
Environment=FINAL_DIR=/path/to/openlist/storage
Environment=MANIFEST_DIR=/opt/file-receiver/manifests
Environment=LOG_PATH=/opt/file-receiver/manifests/uploads.log
Environment=STAGING_TTL_HOURS=48
ExecStart=/usr/bin/python3 -m uvicorn server:app --host 0.0.0.0 --port 8080
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

### 3) Bot (example)

```bash
go run ./cmd
```

## Configuration

### Global (excerpt)

- `stream`: true for streaming; false writes to local temp file (better for resume).
- `workers`: concurrent download tasks.
- `threads`: max threads per task.

### WebDAV / OpenList storage (key)

```toml
[[storages]]
name = "OpenList"
type = "webdav"
enable = true
base_path = "/"
receiver_url = "http://<receiver-host>:8080"
chunk_size_mb = 10
chunk_retries = 3

# Keep these if you still need WebDAV listing/reading:
# url = "https://example.com/dav"
# username = "username"
# password = "password"
```

Notes:
- Set `base_path = "/"` to avoid duplicated subfolders (e.g. `/Tele/Tele`).
- If `receiver_url` is set, upload uses the receiver; otherwise it falls back to direct WebDAV PUT.

### Resume behavior

- **Seekable source (stream=false)**: resumes from existing offset.
- **Non-seekable (stream=true)**: staging is reset and upload restarts.

## Full config.toml example (sanitized)

```toml
# Global
stream = true
workers = 3
threads = 4

[telegram]
token = "YOUR_BOT_TOKEN"
app_id = 123456
app_hash = "YOUR_APP_HASH"

[telegram.userbot]
enable = true
session = "data/usersession.db"

[[storages]]
name = "OpenList"
type = "webdav"
enable = true
base_path = "/"
receiver_url = "http://<receiver-host>:8080"
chunk_size_mb = 10
chunk_retries = 3

# Keep these if you still need WebDAV listing/reading:
# url = "https://example.com/dav"
# username = "username"
# password = "password"

[[users]]
id = 1234567890
storages = ["OpenList"]
blacklist = false
```

## Receiver Environment Variables

- `FINAL_DIR`: target directory (OpenList local storage path)
- `STAGING_DIR`: staging directory for partial uploads
- `MANIFEST_DIR`: where upload manifests are stored
- `LOG_PATH`: append-only upload log
- `STAGING_TTL_HOURS`: auto cleanup threshold

## Receiver Endpoints

- `POST /upload_chunk`: upload chunk (`file`, `filename`, `upload_id`, `offset`)
- `POST /complete`: finalize (`filename`, `upload_id`)
- `GET /status`: query status (`upload_id`)
- `POST /reset`: reset staging for an upload id
- `GET /healthz`: health check

## Tutorial: from zero to OpenList

1. **Prepare directories**
   - Receiver root: `/opt/file-receiver`
   - Staging/manifest: `/opt/file-receiver/staging`, `/opt/file-receiver/manifests`
   - Confirm OpenList local storage path (e.g. `/path/to/openlist/storage`)
2. **Deploy receiver**
   - Docker: edit `file-receiver/docker-compose.yml` and set `FINAL_DIR` mount
   - systemd: edit `file-receiver/file-receiver.service` and set `FINAL_DIR`
3. **Start receiver**
   - Docker: `docker compose up -d --build`
   - systemd: `systemctl enable --now file-receiver`
4. **Verify**
   - `curl http://<receiver-host>:8080/healthz`
5. **Configure bot**
   - Set `receiver_url` in your WebDAV/OpenList storage config
6. **Upload a large file**
   - Send a >100MB file and confirm it appears in OpenList

## Troubleshooting

- **Path duplicated (Tele/Tele)**: check `base_path` and OpenList storage path.
- **Upload fails**: ensure receiver is reachable and not proxied by Cloudflare for large uploads.
- **Files not visible**: verify `FINAL_DIR` and refresh OpenList.

## Security Notes

- Receiver has no built-in auth by default. Keep it on a private network.
- If exposed publicly, add reverse-proxy auth/ACL/rate limit.

## Thanks To

- [gotd](https://github.com/gotd/td)
- [TG-FileStreamBot](https://github.com/EverythingSuckz/TG-FileStreamBot)
- [gotgproto](https://github.com/celestix/gotgproto)
- [tdl](https://github.com/iyear/tdl)
- All the dependencies, contributors and users.
