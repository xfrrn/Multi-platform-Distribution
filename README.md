# Multi-platform Distribution

Desktop app update server written in Go. It manages apps, releases, uploaded artifacts, and exposes a client-facing latest-version API.

## MVP scope

- App management
- Release management
- Local artifact upload with SHA512 calculation
- Generic `update.json` style latest manifest
- Admin login with JWT plus optional API key for automation

## Local setup

1. Start PostgreSQL, the API server, and the web console:

   ```powershell
   docker compose -f deploy/docker-compose.yml up -d postgres update-server web
   ```

   The server runs database migrations automatically when `AUTO_MIGRATE=true`.
   The web console is available at `http://localhost:5173`.

2. For direct local development, copy `.env.example` values into your shell, start PostgreSQL, then run:

   ```powershell
   go run ./cmd/server
   ```

3. Start the web console:

   ```powershell
   cd web
   npm install
   npm run dev
   ```

   Vite serves the console at `http://localhost:5173` and proxies `/api` to `http://localhost:8080`.

## Core endpoints

- `GET /healthz`
- `POST /api/auth/login`
- `POST /api/apps`
- `GET /api/apps`
- `GET /api/apps/:appId`
- `PATCH /api/apps/:appId`
- `DELETE /api/apps/:appId`
- `POST /api/apps/:appId/releases`
- `GET /api/apps/:appId/releases`
- `PATCH /api/releases/:releaseId`
- `DELETE /api/releases/:releaseId`
- `GET /api/releases/:releaseId/artifacts`
- `POST /api/releases/:releaseId/artifacts`
- `GET /api/artifacts/:artifactId`
- `PATCH /api/artifacts/:artifactId`
- `PUT /api/artifacts/:artifactId/file`
- `DELETE /api/artifacts/:artifactId`
- `GET /api/artifacts/:artifactId/download`
- `GET /api/latest/:appSlug?channel=stable&platform=windows&arch=x64&client_id=device-1`
- `GET /api/latest/:appSlug/update.json`
- `GET /api/latest/:appSlug/latest.yml`
- `GET /api/latest/:appSlug/appcast.xml`
- `GET /api/stats/summary`
- `GET /api/stats/update-requests`
- `GET /api/stats/downloads`
- `GET /api/apps/:appId/stats/summary`
- `GET /api/apps/:appId/stats/update-requests`
- `GET /api/apps/:appId/stats/downloads`
- `GET /api/releases/:releaseId/stats`

Delete endpoints use soft archive. Archived apps, releases, and artifacts stay in storage but are hidden from management lists and latest metadata.

Staged releases use `client_id` for stable hash rollout. Requests without `client_id` skip staged releases below 100%.

Stats endpoints support `date_from`, `date_to`, `channel`, `platform`, `arch`, `client_id`, `limit`, and `offset` where applicable.

## Admin authentication

Management endpoints require either:

- `Authorization: Bearer <access_token>` from `POST /api/auth/login`
- `X-API-Key: <API_KEY>` for CI/CD automation when `API_KEY` is configured

Public endpoints are limited to health checks, update metadata, static local downloads, and artifact download redirects.

On first startup, set `BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` to create the first admin user. The bootstrap step is skipped after any admin user exists.

Login example:

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/auth/login `
  -ContentType application/json `
  -Body '{"email":"admin@example.com","password":"change-me-now"}'
```

## Storage

Local storage is the default and serves files from `/downloads`.

Set `STORAGE_DRIVER=s3` or `STORAGE_DRIVER=minio` to use an S3-compatible backend. Required variables are `S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY`, and `S3_SECRET_KEY`. Use `S3_PUBLIC_BASE_URL` when files are served through a CDN, reverse proxy, or public bucket URL.

Anyshare can be enabled as an experimental artifact source without replacing `STORAGE_DRIVER`. Set `ANYSHARE_ENABLED=true`, `ANYSHARE_BASE_URL`, `ANYSHARE_SHARING_LINK`, and `ANYSHARE_UPLOAD_PATH`. Use a larger `ANYSHARE_TIMEOUT`, such as `10m`, for installer uploads. Anyshare artifacts are uploaded to the configured remote folder, but manifests expose the stable server download endpoint so downloads can resolve a fresh Anyshare direct URL at request time.
