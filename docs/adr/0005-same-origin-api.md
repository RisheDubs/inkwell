# ADR-0005: The browser calls the API on its own origin

- **Status:** Accepted
- **Date:** 2026-11-12 (recorded 2026-11-17; referenced in code and the progress log since then)
- **Deciders:** Rishekesh

## Context

The web app and the Go API are separate servers. If the browser called the API on another origin,
every request would need CORS: allowed origins, allowed headers (Authorization), and preflight
requests before most calls. A mistake there either breaks the app or allows other sites in.

## Decision

The browser only ever talks to one origin and requests `/api/...` on it.

- Locally: Vite's dev server proxies `/api` to the Go API (`localhost:8080` on the host,
  `api:8080` inside Compose; ADR-0006).
- In production (Phase 5): CloudFront serves the web app and forwards `/api/*` to the load balancer.

## Consequences

- No CORS configuration on the API, and no preflight requests.
- `apiFetch` refuses anything that isn't an `/api/` path, so the token never goes to another site.
- The S3 bucket still needs CORS, because images upload straight from the browser to S3.
- The app must be opened at `http://localhost:5173`, not `127.0.0.1`: the bucket's CORS allows that exact origin.
- Phase 5 must add the `/api/*` behaviour to CloudFront and the CloudFront domain to the bucket CORS.