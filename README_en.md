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

## Features

- Support documents / videos / photos / stickers… and even [Telegraph](https://telegra.ph/)
- Bypass "restrict saving content" media
- Batch download
- Streaming transfer
- Multi-user support
- Auto organize files based on storage rules
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

## Architecture (Chunked Upload)

1. Bot uploads file in chunks to the receiver (`/upload_chunk`).
2. Receiver writes chunks to staging and records progress.
3. Bot calls `/complete`, receiver moves the file to OpenList local storage.

## Quick Start (Chunked Upload)

### 1) Deploy receiver (Docker)

```bash
cd file-receiver
docker compose up -d --build
```

### 2) Configure storage

```toml
[[storages]]
name = "OpenList"
type = "webdav"
enable = true
base_path = "/"
receiver_url = "http://<receiver-host>:8080"
chunk_size_mb = 10
chunk_retries = 3

# Keep these if you still need WebDAV listing/reading
# url = "https://example.com/dav"
# username = "username"
# password = "password"
```

### 3) Run bot

```bash
go run ./cmd
```

## Receiver Environment Variables

- `FINAL_DIR`: target directory (OpenList local storage path)
- `STAGING_DIR`: staging directory for partial uploads
- `MANIFEST_DIR`: where upload manifests are stored
- `LOG_PATH`: append-only upload log
- `STAGING_TTL_HOURS`: auto cleanup threshold

## Thanks To

- [gotd](https://github.com/gotd/td)
- [TG-FileStreamBot](https://github.com/EverythingSuckz/TG-FileStreamBot)
- [gotgproto](https://github.com/celestix/gotgproto)
- [tdl](https://github.com/iyear/tdl)
- All the dependencies, contributors and users.
