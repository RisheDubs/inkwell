# ADR-0006: The web app runs in Compose as a Vite dev container

- Status: Accepted
- Date: 2026-11-12
- Deciders: Rishekesh

## Context

`docker compose up` must run the whole app, so the web app needs a Compose service. In
production (Phase 5) the web app is static files in S3 behind CloudFront, not a container,
so this container only ever runs locally.

I had two options:

- A. Run the Vite dev server in a container, with the source mounted for live reload.
- B. Build the app and serve it with nginx, like a production image.

## Decision

I use A. The `web` service runs `vite` on port 5173, bound to 127.0.0.1 like the other
services. The source is bind-mounted; `node_modules` lives in its own volume so the
container's Linux packages never mix with the host's. Inside Compose, Vite proxies `/api` to
`http://api:8080` (the service name), set by an environment variable so it still works when
Vite runs on the host.

The production build is still proven on every PR: CI runs `npm run build`.

## Consequences

- Live reload inside Compose; no rebuild to see a change.
- No nginx image to maintain that production would never use.
- Vite must listen on 0.0.0.0 inside the container, or the published port reaches nothing.
- The proxy target differs between host (`localhost:8080`) and Compose (`api:8080`).
- The browser origin stays http://localhost:5173, so the S3 bucket CORS needs no change.
- Security headers (CSP and others) belong to CloudFront in Phase 5, not to this container.