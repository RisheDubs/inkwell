# ADR-0009: Image links come from a resolver the page chooses, with no default

- **Status:** Accepted
- **Date:** 2026-11-17
- **Deciders:** Rishekesh

## Context

Entry bodies and covers store only a media ID; the browser asks the API for a short-lived link
to show each image. Until step 18 there was one way to ask: `getImageUrl`, which calls
`/api/media/{id}` with the signed-in user's token.

Step 18 adds public pages at `/j/{handle}`. Their images must come from
`/api/public/{handle}/media/{id}`, with no token, and must work signed out. The same components
(`MediaImageView` inside the editor or read-only body, and `CoverImage`) draw images on both kinds
of page, so they need to be told which route to use.

Options considered:

- **A. React context with the private resolver as the default.** No change to the signed-in
  pages or their tests. But a public page that forgets to set the resolver would quietly call the
  private route with a signed-in visitor's token. The API's ownership check returns 404, so
  nothing leaks, yet images break without an error and a token is sent where it was never needed.
- **B. React context with no default; reading it without a provider throws.** Every page must
  choose. Forgetting fails at once, in tests.

## Decision

Option B.

- `lib/imageResolver.ts` holds `ImageResolverContext` (default `null`) and `useImageResolver()`,
  which throws if no page set it.
- `RequireAuth` provides `getImageUrl` to every signed-in route.
- `PublicFrame` provides `getPublicImageUrl(handle, id)` to the public routes, memoised on the
  handle so image effects don't re-run on every render.
- `lib/public.ts` never asks for a token and sends no cookies (`credentials: 'omit'`).
- The editor and the read-only body share one node list (`bodyExtensions`), so the public page
  shows exactly what the editor can write, and refuses anything else.

## Consequences

- A page can't show images without saying where they come from; the wrong default can't happen.
- Tests that render image components outside the real router wrap them in `withImages()`
  (`src/test/images.tsx`), which supplies the (mocked) private resolver.
- Tests prove the public pages never call `getImageUrl`, and that `lib/public.ts` never calls
  `getAccessToken`, even when a user is signed in.
- The server's ownership check stays the real protection; this is defence in depth on the client.
- Interview point: "fail closed instead of a convenient default", the same idea as the API's
  uniform 404 for private journals.