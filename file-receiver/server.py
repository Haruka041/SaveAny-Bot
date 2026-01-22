import asyncio
import json
import os
import pathlib
import re
import time

from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware

FINAL_DIR = os.getenv("FINAL_DIR", os.getenv("UPLOAD_DIR", "/app/uploads"))
STAGING_DIR = os.getenv("STAGING_DIR", "/app/staging")
MANIFEST_DIR = os.getenv("MANIFEST_DIR", os.path.join(STAGING_DIR, "manifests"))
LOG_PATH = os.getenv("LOG_PATH", os.path.join(MANIFEST_DIR, "uploads.log"))
STAGING_TTL_HOURS = int(os.getenv("STAGING_TTL_HOURS", "48"))

os.makedirs(FINAL_DIR, exist_ok=True)
os.makedirs(STAGING_DIR, exist_ok=True)
os.makedirs(MANIFEST_DIR, exist_ok=True)

UPLOAD_ID_RE = re.compile(r"^[A-Za-z0-9._-]{1,128}$")

app = FastAPI()
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


def _safe_relpath(name: str) -> str:
    if os.path.isabs(name):
        raise HTTPException(status_code=400, detail="invalid filename")
    normalized = os.path.normpath(name).lstrip("/").lstrip("\\")
    if normalized.startswith("..") or "/.." in normalized or "\\.." in normalized:
        raise HTTPException(status_code=400, detail="invalid filename")
    if normalized in ("", ".", "/"):
        raise HTTPException(status_code=400, detail="invalid filename")
    return normalized


def _safe_dest(root: str, rel: str) -> str:
    root_path = pathlib.Path(root).resolve()
    dest_path = (root_path / rel).resolve()
    if root_path not in dest_path.parents and dest_path != root_path:
        raise HTTPException(status_code=400, detail="invalid filename")
    return str(dest_path)


def _unique_path(dest_path: str) -> str:
    if not os.path.exists(dest_path):
        return dest_path
    base, ext = os.path.splitext(dest_path)
    for i in range(1, 1001):
        candidate = f"{base}_{i}{ext}"
        if not os.path.exists(candidate):
            return candidate
    raise HTTPException(status_code=409, detail="too many conflicts")


def _manifest_path(upload_id: str) -> str:
    return os.path.join(STAGING_DIR, f"{upload_id}.json")


def _load_manifest(upload_id: str) -> dict:
    path = _manifest_path(upload_id)
    if not os.path.exists(path):
        return {}
    try:
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception:
        return {}


def _write_manifest(upload_id: str, data: dict) -> None:
    path = _manifest_path(upload_id)
    tmp = f"{path}.tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False)
    os.replace(tmp, path)


def _append_log(entry: dict) -> None:
    log_dir = os.path.dirname(LOG_PATH)
    if log_dir:
        os.makedirs(log_dir, exist_ok=True)
    with open(LOG_PATH, "a", encoding="utf-8") as f:
        f.write(json.dumps(entry, ensure_ascii=False) + "\n")


async def _cleanup_loop() -> None:
    ttl_seconds = max(STAGING_TTL_HOURS, 1) * 3600
    while True:
        try:
            now = time.time()
            for name in os.listdir(STAGING_DIR):
                if not (name.endswith(".part") or name.endswith(".json")):
                    continue
                path = os.path.join(STAGING_DIR, name)
                if os.path.isdir(path):
                    continue
                age = now - os.path.getmtime(path)
                if age > ttl_seconds:
                    try:
                        os.remove(path)
                    except FileNotFoundError:
                        pass
        except Exception:
            pass
        await asyncio.sleep(1800)


@app.on_event("startup")
async def _startup() -> None:
    app.state.cleanup_task = asyncio.create_task(_cleanup_loop())


@app.on_event("shutdown")
async def _shutdown() -> None:
    task = getattr(app.state, "cleanup_task", None)
    if task:
        task.cancel()


@app.post("/upload_chunk")
async def upload_chunk(
    file: UploadFile = File(...),
    filename: str = Form(...),
    upload_id: str = Form(...),
    offset: int = Form(0),
):
    if not UPLOAD_ID_RE.match(upload_id):
        raise HTTPException(status_code=400, detail="invalid upload_id")

    rel = _safe_relpath(filename)
    stage_path = os.path.join(STAGING_DIR, f"{upload_id}.part")
    current_size = os.path.getsize(stage_path) if os.path.exists(stage_path) else 0
    if offset != current_size:
        return JSONResponse(
            status_code=409,
            content={"ok": False, "expected_offset": current_size},
        )

    size = 0
    with open(stage_path, "ab") as f:
        while True:
            chunk = await file.read(1024 * 1024)
            if not chunk:
                break
            f.write(chunk)
            size += len(chunk)

    await file.close()
    now = time.time()
    manifest = _load_manifest(upload_id)
    created_at = manifest.get("created_at", now)
    updated = {
        "upload_id": upload_id,
        "filename": rel,
        "bytes": current_size + size,
        "created_at": created_at,
        "updated_at": now,
        "status": "uploading",
    }
    _write_manifest(upload_id, updated)
    print(f"Recv chunk for: {rel}, upload_id: {upload_id}, size: {size}")
    return {"ok": True, "bytes": size}


@app.get("/status")
async def status(upload_id: str):
    if not UPLOAD_ID_RE.match(upload_id):
        raise HTTPException(status_code=400, detail="invalid upload_id")
    stage_path = os.path.join(STAGING_DIR, f"{upload_id}.part")
    size = os.path.getsize(stage_path) if os.path.exists(stage_path) else 0
    return {"ok": True, "upload_id": upload_id, "size": size}


@app.post("/reset")
async def reset_upload(upload_id: str = Form(...)):
    if not UPLOAD_ID_RE.match(upload_id):
        raise HTTPException(status_code=400, detail="invalid upload_id")
    stage_path = os.path.join(STAGING_DIR, f"{upload_id}.part")
    if os.path.exists(stage_path):
        os.remove(stage_path)
    manifest_path = _manifest_path(upload_id)
    if os.path.exists(manifest_path):
        os.remove(manifest_path)
    return {"ok": True}


@app.post("/complete")
async def complete_upload(filename: str = Form(...), upload_id: str = Form(...)):
    if not UPLOAD_ID_RE.match(upload_id):
        raise HTTPException(status_code=400, detail="invalid upload_id")

    rel = _safe_relpath(filename)
    stage_path = os.path.join(STAGING_DIR, f"{upload_id}.part")
    if not os.path.exists(stage_path):
        raise HTTPException(status_code=404, detail="staging file not found")

    dest_path = _safe_dest(FINAL_DIR, rel)
    dest_dir = os.path.dirname(dest_path) or FINAL_DIR
    os.makedirs(dest_dir, exist_ok=True)
    dest_path = _unique_path(dest_path)

    os.replace(stage_path, dest_path)
    now = time.time()
    manifest = _load_manifest(upload_id)
    manifest.update(
        {
            "upload_id": upload_id,
            "filename": rel,
            "bytes": os.path.getsize(dest_path),
            "completed_at": now,
            "status": "completed",
            "final_path": dest_path,
        }
    )
    manifest_out = os.path.join(MANIFEST_DIR, f"{upload_id}.json")
    tmp = f"{manifest_out}.tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(manifest, f, ensure_ascii=False)
    os.replace(tmp, manifest_out)
    stage_manifest = _manifest_path(upload_id)
    if os.path.exists(stage_manifest):
        os.remove(stage_manifest)
    _append_log(
        {
            "upload_id": upload_id,
            "filename": rel,
            "bytes": manifest.get("bytes", 0),
            "completed_at": now,
            "final_path": dest_path,
        }
    )
    print(f"Complete: {rel}, upload_id: {upload_id}, path: {dest_path}")
    return {"ok": True, "path": rel}


@app.get("/healthz")
async def healthz():
    return {"ok": True}
