package webdav

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/rs/xid"

	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
)

const (
	defaultChunkSizeMB  = 10
	defaultChunkRetries = 3
)

func (w *Webdav) saveChunked(ctx context.Context, r io.Reader, storagePath string) error {
	storagePath = w.JoinStoragePath(storagePath)
	if w.client != nil {
		ext := path.Ext(storagePath)
		base := strings.TrimSuffix(storagePath, ext)
		candidate := storagePath
		for i := 1; w.Exists(ctx, candidate); i++ {
			candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
			if i > 1000 {
				w.logger.Errorf("Too many attempts to find a unique filename for %s", storagePath)
				candidate = fmt.Sprintf("%s_%s%s", base, xid.New().String(), ext)
				break
			}
		}
		storagePath = candidate
	}
	safePath, err := sanitizeRelativePath(storagePath)
	if err != nil {
		return err
	}

	chunkSizeMB := w.config.ChunkSizeMB
	if chunkSizeMB <= 0 {
		chunkSizeMB = defaultChunkSizeMB
	}
	retries := w.config.ChunkRetries
	if retries <= 0 {
		retries = defaultChunkRetries
	}

	if w.receiverClient == nil {
		w.receiverClient = &http.Client{Timeout: time.Hour * 2}
	}

	chunkURL, completeURL, statusURL, resetURL, err := receiverEndpoints(w.config.ReceiverURL)
	if err != nil {
		return err
	}
	uploadID := uploadIDFromContext(ctx, storagePath)

	var sent int64
	if statusURL != "" {
		if size, err := getStatus(ctx, w.receiverClient, statusURL, uploadID); err == nil && size > 0 {
			if seeker, ok := r.(io.Seeker); ok {
				if _, err := seeker.Seek(size, io.SeekStart); err != nil {
					return fmt.Errorf("failed to seek upload reader: %w", err)
				}
				sent = size
				w.logger.Infof("Resuming upload at offset %d for %s", sent, storagePath)
			} else {
				if resetURL != "" {
					_ = resetUpload(ctx, w.receiverClient, resetURL, uploadID)
				}
				w.logger.Warnf("Upload reader is not seekable; restarting upload for %s", storagePath)
			}
		}
	}

	buf := make([]byte, chunkSizeMB*1024*1024)
	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			var lastErr error
			for attempt := 1; attempt <= retries; attempt++ {
				if err := postChunk(ctx, w.receiverClient, chunkURL, safePath, uploadID, sent, chunk); err != nil {
					lastErr = err
					time.Sleep(time.Duration(attempt) * time.Second)
					continue
				}
				lastErr = nil
				break
			}
			if lastErr != nil {
				return fmt.Errorf("failed to upload chunk at offset %d: %w", sent, lastErr)
			}
			sent += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read upload source: %w", readErr)
		}
	}
	if err := postComplete(ctx, w.receiverClient, completeURL, safePath, uploadID); err != nil {
		return fmt.Errorf("failed to complete upload: %w", err)
	}
	return nil
}

func postChunk(ctx context.Context, client *http.Client, receiverURL, filename, uploadID string, offset int64, chunk []byte) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("filename", filename); err != nil {
		return err
	}
	if err := writer.WriteField("upload_id", uploadID); err != nil {
		return err
	}
	if err := writer.WriteField("offset", fmt.Sprintf("%d", offset)); err != nil {
		return err
	}
	part, err := writer.CreateFormFile("file", path.Base(filename))
	if err != nil {
		return err
	}
	if _, err := part.Write(chunk); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, receiverURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusConflict {
			return fmt.Errorf("offset mismatch")
		}
		return fmt.Errorf("receiver returned %s", resp.Status)
	}
	return nil
}

func postComplete(ctx context.Context, client *http.Client, completeURL, filename, uploadID string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("filename", filename); err != nil {
		return err
	}
	if err := writer.WriteField("upload_id", uploadID); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, completeURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("receiver returned %s", resp.Status)
	}
	return nil
}

func getStatus(ctx context.Context, client *http.Client, statusURL, uploadID string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL+"?upload_id="+uploadID, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("receiver returned %s", resp.Status)
	}
	var payload struct {
		Size int64 `json:"size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	return payload.Size, nil
}

func resetUpload(ctx context.Context, client *http.Client, resetURL, uploadID string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("upload_id", uploadID); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resetURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("receiver returned %s", resp.Status)
	}
	return nil
}

func receiverEndpoints(base string) (string, string, string, string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", "", "", "", fmt.Errorf("receiver_url is empty")
	}
	if strings.HasSuffix(base, "/upload_chunk") {
		base = strings.TrimSuffix(base, "/upload_chunk")
		base = strings.TrimRight(base, "/")
		return base + "/upload_chunk", base + "/complete", base + "/status", base + "/reset", nil
	}
	base = strings.TrimRight(base, "/")
	return base + "/upload_chunk", base + "/complete", base + "/status", base + "/reset", nil
}

func uploadIDFromContext(ctx context.Context, storagePath string) string {
	if length := ctx.Value(ctxkey.ContentLength); length != nil {
		if l, ok := length.(int64); ok && l > 0 {
			h := sha1.Sum([]byte(fmt.Sprintf("%s:%d", storagePath, l)))
			return hex.EncodeToString(h[:8])
		}
	}
	return xid.New().String()
}

func sanitizeRelativePath(p string) (string, error) {
	clean := path.Clean(p)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." {
		return "", fmt.Errorf("invalid storage path")
	}
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, "/..") {
		return "", fmt.Errorf("invalid storage path: %s", clean)
	}
	return clean, nil
}
