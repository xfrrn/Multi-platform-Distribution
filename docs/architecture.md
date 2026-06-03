# Architecture

The project starts as a modular monolith:

- `cmd/server`: process entrypoint.
- `internal/app`: dependency wiring and lifecycle.
- `internal/http`: Gin router, handlers, middleware.
- `internal/domain`: core data structures.
- `internal/service`: business workflows and validation.
- `internal/repository/postgres`: PostgreSQL persistence.
- `internal/storage`: object storage abstraction, local implementation, and S3-compatible implementation.
- `internal/metadata`: generic update manifest, Electron `latest.yml`, and appcast XML rendering.
- `internal/auth`: JWT issuing and verification.
- `internal/database`: embedded SQL migrations.

The first development phase focuses on the release pipeline:

1. Create an app.
2. Create a release.
3. Upload artifacts.
4. Generate a client-facing latest manifest.

Detailed statistics, generated blockmap support, and richer admin roles are planned as later modules.
