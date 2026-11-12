# ADR-0007: Functional UI first, redesign after the project is complete

- Status: Accepted
- Date: 2026-11-12
- Deciders: Rishekesh

## Context

Phase 4 builds the frontend: auth, timeline, editor, uploads, theming and the public share
page. I plan to redesign the look once all phases are done, so polishing the visuals now would
be work I throw away. The parts that carry the risk (auth, token handling, uploads, XSS
safety, tests, the container and CI) are not visual and won't change in a redesign.

## Decision

Phase 4 ships a plain, functional UI with Tailwind utility classes and no design system.

- All API calls, auth and upload logic live in `src/lib/`. Components only call those
  functions and render, so a redesign replaces components and styles, not logic.
- Theming (paper, font, accent, cover) still works, because users styling their journal is a
  core feature; it's built on CSS variables, so a new design can reuse it.
- The redesign will use GetLayers (connected as an MCP) to pick one Style and section layouts,
  then apply them to these components. Anything it generates that costs credits needs my
  approval first.

## Consequences

- Phase 4 time goes into behaviour and security, not visuals.
- The app looks basic in early demos; screenshots for the README wait for the redesign.
- Keeping logic out of components is a rule to hold in review: a `fetch` call or token access
  inside a component is a bug.
- The redesign gets its own branch and ADR, and must keep every Phase 4 test passing.