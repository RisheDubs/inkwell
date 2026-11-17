# ADR-0008: Public data is never derived from private data

- **Status:** Accepted
- **Date:** 2026-11-15
- **Deciders:** Rishekesh

## Context

Step 17 added public journals at `/api/public/{handle}`. The public response types leave out
every private field (owner ID, email, storage keys, timestamps), and a test checks for them.

The real-stack isolation test still found a leak: a new journal's default title was built as
`displayName + "'s Journal"`, and the display name can start out as part of the user's email
address. Making a journal public with its default title would publish part of the owner's email.
The field itself (title) was meant to be public; the value inside it was derived from private data.

## Decision

- New journals get the fixed title **"My Journal"**. `NewJournal` no longer takes a display name,
  so the old behaviour can't come back by accident.
- General rule: a value that can become public must never default to, or be built from, private
  data (email, IDs, keys, timestamps). Defaults are fixed strings or chosen by the user.
- Journals still start **private**; a unit test pins both the title and the visibility.

## Consequences

- Existing development journals keep their old titles until renamed in Settings.
- Step 18's visibility switch should remind the owner that the title and theme become public.
- Leak tests on public responses stay in place; they check derived values, not just field names.