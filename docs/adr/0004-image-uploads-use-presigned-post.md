# ADR-0004: Image uploads use presigned POST and server-built keys

- Status: Accepted
- Date: 2026-11-11
- Deciders: Rishekesh

## Context

Images go straight from the browser to S3, so the API never streams the bytes. The build plan
said presigned PUT URLs with a 5-minute expiry, image/* only and a 5 MB limit. While building it
I found that a presigned PUT can't enforce a size range: S3 accepts whatever size the client sends.
The limit would only be a promise in our docs, not a rule anyone enforces.

## Decision

I use a presigned POST instead. Its signed policy lists conditions S3 checks itself: the exact key,
the exact Content-Type I approved (jpeg, png, webp or gif), and content-length-range 1 to 5 MB.
The link expires after 5 minutes. The key is always users/{userId}/media/{24-hex-id}.{ext}, built
from the token's user ID, never from the request. Reads use 15-minute presigned GET links built
the same way, so a user can only ever get a link into their own folder.

There is no media collection yet: a media ID plus the caller's user ID is enough to find the object.

## Consequences

- The 5 MB limit is enforced by S3, not by trust. A tampered form fails the signature check.
- The browser sends a multipart form instead of a single PUT. That's a few more lines in Phase 4.
- The API can't list a user's images or tell whether an upload finished. If Phase 4 needs that,
  I'll add a media collection and record it here.
- A link signed with my 1-hour local credentials stops working when they expire, even before its
  own expiry. ECS task-role credentials refresh automatically, so production isn't affected.

## Alternatives considered

- **Presigned PUT (the plan).** Simpler client, but no enforced size limit. Rejected.
- **Upload through the API.** The API could check every byte, but it would carry every image's
  traffic and memory. Rejected for now.