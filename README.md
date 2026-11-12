# Inkwell

A private, customisable journal app. Each user gets their own notebook-style journal, with optional AI that helps them write and reflect.

> **Work in progress.** Built in phases as a learning and portfolio project. Current phase: **Phase 1, API skeleton + MongoDB in Docker**.

## Tech stack (planned)

| Layer | Choice |
|---|---|
| Backend | Go, chi router, slog structured logging |
| Frontend | React, Vite, TypeScript, Tailwind, Tiptap editor |
| Database | MongoDB 8 (`mongodb-atlas-local` in Docker; Atlas M0 in production) |
| Storage | Amazon S3 with SSE-KMS, Block Public Access |
| Auth | Amazon Cognito |
| AI | Amazon Bedrock (Titan embeddings + a Claude model), opt-in only |
| Infrastructure | Docker, ECS Fargate, CloudFront, Terraform |
| CI/CD | GitHub Actions with AWS OIDC (no stored keys) |

## Repository layout

```
apps/api/     Go API (cmd/ entry point, internal/ packages)
apps/web/     React frontend
infra/        Terraform modules and environments
deploy/helm/  Helm chart (Kubernetes stretch goal)
docs/         Architecture decision records (ADRs) and progress log
```

## Roadmap

- [x] Phase 0: Tooling, accounts and repository setup
- [x] Phase 1: API skeleton + MongoDB in Docker
- [x] Phase 2: Authentication with Cognito
- [x] Phase 3: Journal core + S3
- [ ] Phase 4: Frontend + customisation
- [ ] Phase 5: AWS infrastructure with Terraform
- [ ] Phase 6: CI/CD
- [ ] Phase 7: AI features
- [ ] Phase 8: Observability, security and polish
- [ ] Phase 9 (stretch): Kubernetes

## Getting started

Coming in Phase 1: `docker compose up` will start the API and MongoDB locally.

**Tested with:** Go 1.27.3 · Docker 29.9.2 · Docker Compose v5.6.1 · Node.js 26.13.0 · AWS CLI 2.39.3 · Terraform 1.17.1, on Windows 11 + WSL2 (Ubuntu 26).

## Security principles

- No secrets in Git: local config in `.env` (git-ignored); production secrets in AWS Secrets Manager.
- Every database query filters by the owner's user ID; S3 is private, with short-lived presigned URLs.
- Least-privilege IAM for every service role.

## License

[MIT](LICENSE)
