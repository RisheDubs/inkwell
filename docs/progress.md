# Progress log

## 2026-11-03 to 04: Phase 0 (setup)

**Built:** WSL2 + Ubuntu 26, Git, Docker Desktop, Go 1.27.3, Node 26 LTS, AWS CLI 2.39.3, Terraform 1.17.1, VS Code (WSL), MongoDB Compass. GitHub repo with folder structure, README, MIT licence and ADR-0001. AWS IAM admin user with MFA, `aws login` for the CLI (no access keys), US$25 budget alerts. Atlas account.

**Learned:** WSL2 vs WSL1, Linux basics, Git history is permanent (rotate leaked secrets first), containers vs VMs, images vs containers, the Docker socket is root-equivalent, account vs identity in AWS (checked with `aws sts get-caller-identity`).

**Open questions:** Did the root account ever have access keys? (Check once.)

## 2026-11-05 to 08: Phase 1 (API skeleton + MongoDB in Docker)

**Built:** Go API with chi, `/healthz` that pings MongoDB (503 when it's down), config from environment variables, JSON logs with slog, graceful shutdown. MongoDB 8.3.9 (atlas-local) in Compose with auth, localhost-only ports and named volumes. Multi-stage Dockerfile on distroless, non-root: 13 MB image. Makefile. CI: lint, race-enabled tests, Docker build. `main` protected by a ruleset.

**Learned:** Go modules, handlers, interfaces and dependency injection, table-driven tests, contexts and timeouts, goroutines and channels, layer caching, service names as hostnames, `depends_on` health gating, pinning actions to SHAs.

**Measured:** 13 MB image; cold build 2m 53s, cached 3.2s; 0 lint issues; config 100% and httpapi 93.3% coverage.

**Open questions:** Align the MongoDB image version with Atlas M0 (Phase 5).

## 2026-11-09: Phase 2 (authentication with Cognito)

**Built:** Cognito user pool and app client (Sydney). Access-token verification against Cognito's JWKS (RS256, issuer, token_use, client_id, expiry). `GET /api/me` creates the user and their journal on first call, with the email from Cognito GetUser. `PATCH /api/me` with validation and unique handles. One error format with request IDs, per-request logs, 4 KB body limit, unknown fields rejected. MongoDB integration tests in CI. ADR-0002.

**Learned:** OAuth 2.0 and OIDC basics, JWT structure, JWKS, why ID tokens aren't for APIs, upserts with `$setOnInsert`, partial unique indexes, build tags, middleware and route groups.

**Measured:** 11 forged or bad-token cases rejected in tests; coverage config 100%, httpapi 94.0%, auth 84.8%, journal 81.6%; 4 integration tests passing in 4.99s; first sign-in 1,133 ms (includes the Cognito call), later requests 1 ms, rejected tokens under 1 ms.

**Open questions:** Rate-limit auth-heavy routes (Phase 7/8). Turn off `ALLOW_USER_PASSWORD_AUTH` (Phase 5).

## Phase 3: Journal, entries, S3 and KMS (9 to 11 Nov 2026)

### What we built
- 3A: GET/PATCH /api/journal (theme allow-lists, dot-notation updates); entry repository with
  cursor pagination (newest first), tag filter, soft delete and owner-scoped queries.
- 3B: KMS customer-managed key, private S3 bucket (Block Public Access, TLS-only policy, SSE-KMS +
  Bucket Key) and a least-privilege role inkwell-dev-api; /api/entries CRUD with Tiptap bodies in S3;
  server-computed excerpt and word count; local API runs on 1-hour role credentials (ADR-0003).
- 3C: presigned POST image uploads (5 min, jpeg/png/webp/gif, 5 MB enforced by S3, server-built keys)
  and 15-minute download links (ADR-0004); two-user HTTP isolation tests in CI; timestamps stored and
  returned to the millisecond.

### Measured
- Image 17.8 MB (13 MB before the AWS SDK).
- Coverage: config 100%, store/s3 <...>%, httpapi <...>%, journal <...>%, auth 84.8%.
- IAM probes as the role: 1 allowed, 3 denied (write outside users/, list bucket, direct KMS), object
  encrypted with our key.
- End to end against real AWS: 10 entry checks and 10 upload checks passed.

### What I learned
- Explicit deny beats allow, and allow beats implicit deny; a TLS-only bucket policy blocks even admins.
- Without s3:ListBucket, a missing object is a 403, not a 404.
- A presigned PUT can't enforce a size limit; a presigned POST policy can.
- Docker Compose takes a variable from the shell before .env, so load .env in a subshell.
- Rotate a credential on the server first, then in the config; health checks notice first.
- A secret leaks the moment it's in a terminal, screenshot, chat or log, not only on GitHub.

### Next step
Phase 4: React + Vite + Tiptap frontend with Amplify sign-in.