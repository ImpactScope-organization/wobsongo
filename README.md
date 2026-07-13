# Wobsongo

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![DPG Candidate](https://img.shields.io/badge/DPG-Candidate-green)](https://digitalpublicgoods.net/)
[![Go](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev/)

> **⚠️ This project is under active development and is not yet production-ready.**

**Wobsongo** is an open-source backend engine for misinformation detection, powered by a knowledge management and hybrid retrieval system. It provides the infrastructure to ingest content, build a structured knowledge base, and verify claims against it.

The backend is written in Go and exposes a REST API for document processing and claim verification workflows. It uses an async job queue to handle long-running tasks reliably, and supports S3-compatible object storage for document and media assets.

## Architecture

- **REST API** — [Echo](https://echo.labstack.com/) web framework
- **Async Job Queue** — [River](https://riverqueue.com/) backed by PostgreSQL
- **Database** — PostgreSQL 18
- **Object Storage** — S3-compatible (MinIO for local development)

## SDG Alignment

- **SDG 3 (Good Health & Well-being)** — Infrastructure to combat health misinformation
- **SDG 16 (Peace, Justice, Strong Institutions)** — Transparent, auditable fact-checking

## Prerequisites

- [Docker](https://docs.docker.com/get-started/get-docker/) — for local infrastructure via Docker Compose
- Go 1.26+ — install manually or via a version manager of your choice

We recommend using [Jetify Devbox](https://www.jetify.com/docs/devbox/quickstart/) to keep your local toolchain consistent with the rest of the team. Devbox pins the exact versions of Go, golangci-lint, and other tools declared in `devbox.json`, so your environment matches CI without any manual setup.

## Quick Start

```bash
# 1. Clone
git clone https://github.com/ImpactScope-organization/wobsongo.git
cd wobsongo

# 2. (Recommended) Enter the Devbox shell — pins Go and tooling versions automatically
devbox shell

# 3. Copy and configure environment variables
cp .env.example .env
# Edit .env with your settings

# 4. Start infrastructure (PostgreSQL, MinIO, MailHog)
make dbup

# 5. Run database migrations
make migrateup

# 6. Start the server with hot-reload
make dev
```

The API will be available at `http://localhost:8000`.

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `APP_DB_URI` | ✅ | — | PostgreSQL connection string |
| `APP_JWT_SECRET` | ✅ | — | JWT signing key |
| `APP_JWT_EXPIRY_HOURS` | ✅ | — | JWT token expiry in hours |
| `APP_ENV` | | `development` | Environment (`development`, `staging`, `production`) |
| `APP_PORT` | | `8000` | Server port |
| `APP_LOG_LEVEL` | | `1` | Log level (0=debug, 1=info, 2=warn, 3=error) |
| `API_HOST` | | `localhost:8000` | Public API hostname |
| `FRONTEND_HOST` | | `localhost:5173` | Frontend hostname for CORS |
| `CORS_ALLOWED_ORIGINS` | | `*` | Comma-separated list of allowed CORS origins |
| `STORAGE_PROVIDER` | | `local` | Storage backend (`local`, `s3`) |
| `EMAIL_PROVIDER_TRANSACTIONAL` | | `mock` | Email backend (`mock`, `mailhog`, `smtp`) |

See `.env.example` for the full list including S3 and SMTP options.

## Development Commands

```bash
make fmt          # Format code
make check        # Lint and build
make test-unit    # Run unit tests
make test         # Run full test suite (requires make dbtestup first)
make gen          # Regenerate code (mocks, swagger docs)
```

See `Makefile` for all available targets.

## Contributing

We welcome contributions from developers, NGOs, and researchers. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

- Report bugs via [GitHub Issues](https://github.com/ImpactScope-organization/wobsongo/issues)
- For security vulnerabilities, see [SECURITY.md](SECURITY.md) — do not open a public issue
- Full documentation: [docs site](https://impactscope-organization.github.io/wobsongo)

## License

Distributed under the [Apache License 2.0](LICENSE). This license was chosen deliberately to maximize adoption by governments, NGOs, and private sector partners without legal friction.

## Acknowledgements

Wobsongo is developed by [ImpactScope](https://impactscope.com) as the technical partner for [KAIROS](https://kairos-africa.org), with support from UNICEF.
