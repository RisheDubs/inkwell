# ADR-0010: Terraform layout, remote state, and a runtime stack that is only up for tests

- **Status:** Accepted
- **Date:** 2026-11-18
- **Deciders:** Rishekesh

## Context

Phase 5 moves the AWS side of Inkwell into Terraform and deploys the app. The main constraint is
cost: the deployed app (ALB, Fargate task, public IPv4 addresses) costs roughly US$0.045/hour,
about US$33/month if left on, which is over the US$25 budget. The app only needs to run while
testing or demoing.

Terraform also needs somewhere safe to keep state. State records every resource it manages and can
hold secrets (the Atlas password, the Mongo URI) in plain text, so it can't live in git or only on
one laptop.

Options considered for layout:

- **A. One root per environment.** Simple, but switching the app off means destroying the whole
  environment, including the database, Cognito users and stored entries.
- **B. Split prod into `foundation` (always on) and `runtime` (on only for tests).** Two applies
  to bring prod up, but switching off loses no data.

## Decision

Option B, with this layout under `infra/terraform/`:

- `bootstrap/` creates the state bucket `inkwell-tfstate-<account>`: versioned, SSE-S3, TLS-only,
  Block Public Access, ACLs disabled, `prevent_destroy`, old versions expire after 90 days. It was
  applied with local state, then its own state was migrated into the bucket
  (`bootstrap/terraform.tfstate`).
- Every root uses the S3 backend with `use_lockfile = true` (S3 native locking, Terraform 1.10+),
  so there is no DynamoDB lock table.
- `modules/` holds reusable modules (network, ecr, storage, cognito, atlas, ecs, cdn).
- `envs/dev/` manages the dev resources created by hand in Phases 2-3 (see ADR-0011).
- `envs/prod/foundation/` is always on and costs about US$1.50/month: network, ECR, content
  bucket and KMS key, Cognito, Atlas M0, the Mongo URI secret, and the web bucket.
- `envs/prod/runtime/` is applied for a test session and destroyed afterwards: ALB, ECS service,
  CloudFront, and the content bucket's CORS rule for the CloudFront domain. `runtime` reads
  `foundation`'s outputs through `terraform_remote_state`.
- Credentials come from `aws configure export-credentials --profile inkwell --format env`, so no
  keys are written to disk or into Terraform files.

This adds `envs/prod/` next to the plan's `envs/dev/`.

## Consequences

- Turning the app off is one `terraform destroy` on `runtime`; entries, users and images survive.
- The public URL is the CloudFront default domain, so it changes every time `runtime` is
  recreated. That is acceptable for tests; a stable URL would need a domain.
- Bringing `runtime` up or down takes about 10-15 minutes, mostly CloudFront.
- The ALB is billed per started hour, so short sessions still cost a full hour.
- Forgetting to destroy `runtime` is the main cost risk; the budget alert is the backstop.
- The state bucket is never destroyed by Terraform. Removing it is a manual, deliberate step.
- CI runs `fmt`, `init -backend=false`, `validate` and `tflint` with no AWS credentials.
- Interview point: separating state by how often things change (and what they cost) keeps the blast
  radius of a `destroy` small.