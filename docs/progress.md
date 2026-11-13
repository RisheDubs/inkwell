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

## 2026-11-12 · Phase 4, session 1 (steps 1-11 of 24)

**Built** (branch `feat/phase-4-frontend`, 9 commits, latest `4856b17`)
- ADR-0005 same-origin API (Vite proxy now, CloudFront later), ADR-0006 Vite dev container, ADR-0007 functional UI first.
- `apps/web`: Vite + React + TypeScript, Tailwind v4, Vitest + Testing Library + ESLint, validated Cognito config, Amplify Auth.
- React Router with a shared layout; sign-up + email confirmation, sign-in + sign-out, `RequireAuth` route guard, password reset. All checked in the browser against real Cognito.

**Measured**
- `npm install`: 160 packages in 57s. `npm audit --omit=dev`: 0 vulnerabilities (10 moderate, dev-only, minimatch).
- Production build 223 ms. CSS 6.82 kB (2.07 kB gzip). JS 219.75 kB (68.70 kB gzip) before Amplify, 281.39 kB (88.79 kB gzip) after.
- Frontend tests: 47 in 11 files, 4.30s; lint and type check clean.

**Learned**
- Same origin vs CORS; CORS is a browser rule, not auth.
- `VITE_` variables ship to every visitor, so only identifiers go there, never secrets.
- SRP keeps the password in the browser; the API only accepts access tokens, never ID tokens.
- A route guard is UX only; the API's token check is the security.
- Open redirects (only same-app return paths) and user enumeration (same message for real and unknown emails).
- Tokens live in localStorage, so XSS would mean stolen sessions (matters for the editor and the CSP).
- Debugging story: a test failed after its last line ran. Logging inside the test and the JSON report's stack showed `beforeEach(() => mock.mockReset())` was returning the mock, which Vitest ran as cleanup. Fix: braces, so `beforeEach` returns nothing.
- Run commands from the right folder: npm in `apps/web`, git/make at the repo root.

**Open questions**
- `943026b` and `f0873a4` reached `main` without a PR number, but `protect-main` has an empty bypass list. Check at close-out.
- Phase 5: CloudFront `/api/*` behaviour, SPA fallback to `index.html`, CloudFront domain in bucket CORS, remove `ALLOW_USER_PASSWORD_AUTH` and `ALLOW_USER_AUTH`.
- Step 17: public images need a route that finds the owner from the handle.

## 2026-11-13 · Phase 4, session 2 (steps 12-16 of 24)

**Built**
- API client: token only on /api/ paths, the API error shape turned into ApiError, request IDs in messages.
- Timeline: cursor pagination, tag filter, race-safe "Load more", time-zone-safe dates.
- Tiptap editor: create, edit and delete; PATCH sends only changed fields; tag input; inline delete confirmation (soft delete, honest wording).
- Image uploads straight to S3; the body stores only the media ID; a fresh link is fetched on display.
- Theming: paper, font and accent as CSS variables, re-checked before reaching CSS; a Settings page.
- Go: journal cover by media ID; the S3 key is built from the token's user and never returned to clients.
- Fixed: the S3 bucket had no CORS rule (planned in 3C, never applied). Now in infra/dev/s3-cors.json.

**Measured**
- Frontend tests: 135 in 25 files, 6.64s. Go: all packages pass with integration tests; lint 0 issues.
- First /api/me for a new web user: 1,390 ms, then 1 ms.

**Learned**
- CORS blocks reading the response, not sending the request: S3 stored the file (204) while the browser showed a failure. curl ignores CORS, so only a real browser found the missing rule.
- A Vitest beforeEach that returns a function gets that function run as cleanup.
- React warnings in passing tests can hide real bugs: Tiptap's empty-document update on unreadable content looked like the user deleting their text.
- Treat stored data as untrusted on the way out too: entry bodies and theme values are re-checked before rendering.
- Hide internal identifiers: clients get media IDs, never S3 keys (Theme.MarshalJSON).

**Open questions**
- Browser checks not yet reported for 14c/14d, 15d (F5, no Authorization header on S3) and 16b/16d.
- Orphaned S3 objects (old covers, removed images, the CORS-failure upload): the ADR-0004 clean-up item.

**Next step**: step 17, the Go public routes for the share page (written test-first).