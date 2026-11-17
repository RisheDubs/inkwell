# ADR-0002: The API accepts Cognito access tokens only

- **Status:** Accepted
- **Date:** Phase 2 (recorded 2026-11-17; the decision was made and built earlier but not written down)
- **Deciders:** Rishekesh

## Context

Cognito issues two JWTs at sign-in. The ID token describes the user (email, name) and is meant for
the client. The access token says what the bearer may do and is meant for APIs. Both are signed by
the same user pool, so an API that only checks the signature would accept either.

## Decision

- The Go API verifies every request's `Authorization: Bearer` token against the pool's JWKS:
  RS256 signature, issuer, expiry, `token_use` must be `access`, and `client_id` must be our app
  client. ID tokens are rejected.
- The user ID comes from the token's `sub`, never from the request body.
- The email is fetched once from Cognito (`GetUser`) when `/api/me` first creates the user.
- The web app sends the access token (`auth.ts`), and a test proves it never sends the ID token.

## Consequences

- A token issued for a different app client, or an ID token, gets a 401.
- The API doesn't trust profile data inside a token; it asks Cognito.
- Tests cover missing, expired and tampered tokens.