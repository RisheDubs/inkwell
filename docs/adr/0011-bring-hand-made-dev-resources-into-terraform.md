# ADR-0011: Bring the hand-made dev resources into Terraform; build prod new

- **Status:** Accepted
- **Date:** 2026-11-18
- **Deciders:** Rishekesh

## Context

In Phases 2-3 these were created by hand in the AWS console for local development:

- Cognito user pool `ap-southeast-2_hinNwGJuW` and app client `inkwell-web`
- S3 bucket `inkwell-dev-content-<account>` (with CORS from `infra/dev/s3-cors.json`)
- KMS key `alias/inkwell-dev-content`
- IAM role `inkwell-dev-api`

They hold real dev data: users, entries and images.

Options considered:

- **A. Import them into `envs/dev` and build prod new from the same modules.** Dev becomes
  reproducible and drift shows up in `plan`. Costs about an hour of `import` work, and a careless
  `destroy` in dev could delete real data.
- **B. Leave dev hand-made and use Terraform for prod only.** Fastest, but dev can't be rebuilt
  from code, and its settings can drift from prod without anyone noticing.
- **C. Reuse the dev resources as prod.** No new resources, but test data and prod data mix, and
  the dev role's trust policy doesn't fit ECS.

## Decision

Option A.

- `envs/dev` uses `import` blocks (Terraform 1.5+) so the import is reviewed in a `plan` first.
  The goal is a plan with no changes, or only additions like tags.
- The dev bucket, KMS key and user pool get `lifecycle { prevent_destroy = true }`.
- Prod gets its own pool, bucket, KMS key and roles, built from the same modules.
- `infra/dev/s3-cors.json` is deleted once its rule is in Terraform.

## Consequences

- Dev and prod come from the same modules, so a difference between them is visible in code.
- Costs are unchanged by the import: the dev KMS key was already US$1/month; prod adds one more.
- `prevent_destroy` means removing dev on purpose needs a code change first, which is intended.
- Interview point: `import` blocks bring existing infrastructure under IaC without recreating it,
  and the plan proves nothing changes.
## Amendment (2026-11-19): dev uses the shared storage module

- `envs/dev` now builds its bucket and key from `modules/storage` (`versioning = false`), the same
  module as prod. `moved` blocks shifted the 9 imported resources into the module in state only;
  the plan was 0 to add, 0 to destroy (one state-only change: `deletion_window_in_days = 7`).
- Modules don't use `prevent_destroy`, because prod must be destroyable in one command. Dev is
  still protected by AWS itself: a non-empty bucket can't be destroyed while `force_destroy` is
  false, a KMS key deletion is only scheduled (7-day window, cancellable), and the user pool has
  `deletion_protection = "ACTIVE"`.
- Dev's CORS rule stays in the dev root, because it is dev-specific (`http://localhost:5173`).
